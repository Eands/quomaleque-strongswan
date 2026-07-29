package radius

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"golang.org/x/crypto/bcrypt"

	"vpn-radius/internal/db"
)

const (
	AccessRequest      = 1
	AccessAccept       = 2
	AccessReject       = 3
	AccountingRequest  = 4
	AccountingResponse = 5
)

const (
	AttrUserName         = 1
	AttrUserPassword     = 2
	AttrFramedIPAddress  = 8
	AttrAcctStatusType   = 40
	AttrAcctInputOctets  = 42
	AttrAcctOutputOctets = 43
	AttrAcctSessionID    = 44
)

type RadiusAttribute struct {
	Type   byte
	Length byte
	Value  []byte
}

type RadiusPacket struct {
	Code          byte
	Identifier    byte
	Length        uint16
	Authenticator [16]byte
	Attributes    []RadiusAttribute
}

type Server struct {
	db     *db.Database
	secret string
	authConn *net.UDPConn
	acctConn *net.UDPConn
}

func NewServer(database *db.Database, secret string) *Server {
	return &Server{
		db:     database,
		secret: secret,
	}
}

func (s *Server) StartAuth(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve auth addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen auth udp: %w", err)
	}
	s.authConn = conn

	log.Printf("RADIUS authentication server listening on %s", addr)

	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("auth read: %w", err)
		}

		packet := parsePacket(buf[:n])
		if packet == nil {
			continue
		}

		clientIP := remoteAddr.IP.String()
		log.Printf("RADIUS auth: code=%d id=%d from %s", packet.Code, packet.Identifier, clientIP)

		if packet.Code == AccessRequest {
			s.handleAccessRequest(packet, clientIP, conn, remoteAddr)
		}
	}
}

func (s *Server) StartAcct(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve acct addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen acct udp: %w", err)
	}
	s.acctConn = conn

	log.Printf("RADIUS accounting server listening on %s", addr)

	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("acct read: %w", err)
		}

		packet := parsePacket(buf[:n])
		if packet == nil {
			continue
		}

		clientIP := remoteAddr.IP.String()
		log.Printf("RADIUS acct: code=%d id=%d from %s", packet.Code, packet.Identifier, clientIP)

		if packet.Code == AccountingRequest {
			s.handleAccountingRequest(packet, clientIP, conn, remoteAddr)
		}
	}
}

func (s *Server) Close() {
	if s.authConn != nil {
		s.authConn.Close()
	}
	if s.acctConn != nil {
		s.acctConn.Close()
	}
}

func (s *Server) handleAccessRequest(packet *RadiusPacket, clientIP string, conn *net.UDPConn, addr *net.UDPAddr) {
	username := getAttrString(packet, AttrUserName)
	password := getAttrString(packet, AttrUserPassword)

	if username == "" {
		s.sendReject(packet, conn, addr)
		s.db.InsertRadiusLog("", "Access-Request", "Reject", clientIP)
		return
	}

	decryptedPassword := decryptPassword(password, s.secret, packet.Authenticator[:])

	user, err := s.db.GetUserByUsername(username)
	if err != nil || !user.IsActive {
		log.Printf("RADIUS: auth failed for %s: %v", username, err)
		s.sendReject(packet, conn, addr)
		s.db.InsertRadiusLog(username, "Access-Request", "Reject", clientIP)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(decryptedPassword)); err != nil {
		log.Printf("RADIUS: bad password for %s", username)
		s.sendReject(packet, conn, addr)
		s.db.InsertRadiusLog(username, "Access-Request", "Reject", clientIP)
		return
	}

	log.Printf("RADIUS: auth success for %s", username)
	s.sendAccept(packet, conn, addr)
	s.db.InsertRadiusLog(username, "Access-Request", "Accept", clientIP)
}

func (s *Server) handleAccountingRequest(packet *RadiusPacket, clientIP string, conn *net.UDPConn, addr *net.UDPAddr) {
	username := getAttrString(packet, AttrUserName)
	acctStatusType := getAttrInt(packet, AttrAcctStatusType)
	sessionID := getAttrString(packet, AttrAcctSessionID)
	framedIP := getAttrString(packet, AttrFramedIPAddress)

	switch acctStatusType {
	case 1:
		s.db.InsertSessionLog(username, framedIP, sessionID)
		log.Printf("RADIUS: accounting start for %s, session %s, IP %s", username, sessionID, framedIP)
	case 2:
		inputOctets := getAttrInt(packet, AttrAcctInputOctets)
		outputOctets := getAttrInt(packet, AttrAcctOutputOctets)
		s.db.UpdateSessionStop(sessionID, int64(inputOctets), int64(outputOctets))
		log.Printf("RADIUS: accounting stop for %s, session %s", username, sessionID)
	case 3:
		inputOctets := getAttrInt(packet, AttrAcctInputOctets)
		outputOctets := getAttrInt(packet, AttrAcctOutputOctets)
		s.db.UpdateSessionInterim(sessionID, int64(inputOctets), int64(outputOctets))
		log.Printf("RADIUS: accounting interim for %s, session %s", username, sessionID)
	}

	s.sendAccountingResponse(packet, conn, addr)
	s.db.InsertRadiusLog(username, "Accounting-Request", "OK", clientIP)
}

func parsePacket(data []byte) *RadiusPacket {
	if len(data) < 20 {
		return nil
	}

	length := binary.BigEndian.Uint16(data[2:4])
	if int(length) > len(data) || int(length) < 20 {
		return nil
	}

	packet := &RadiusPacket{
		Code:       data[0],
		Identifier: data[1],
		Length:     length,
	}
	copy(packet.Authenticator[:], data[4:20])

	offset := 20
	for offset < int(length) {
		if offset+2 > int(length) {
			break
		}

		attrType := data[offset]
		attrLen := data[offset+1]

		if attrLen < 2 || offset+int(attrLen) > int(length) {
			break
		}

		var value []byte
		if attrLen > 2 {
			value = make([]byte, attrLen-2)
			copy(value, data[offset+2:offset+int(attrLen)])
		}

		packet.Attributes = append(packet.Attributes, RadiusAttribute{
			Type:   attrType,
			Length: attrLen,
			Value:  value,
		})

		offset += int(attrLen)
	}

	return packet
}

func getAttrString(packet *RadiusPacket, attrType byte) string {
	for _, attr := range packet.Attributes {
		if attr.Type == attrType {
			return string(attr.Value)
		}
	}
	return ""
}

func getAttrInt(packet *RadiusPacket, attrType byte) uint32 {
	for _, attr := range packet.Attributes {
		if attr.Type == attrType {
			if len(attr.Value) >= 4 {
				return binary.BigEndian.Uint32(attr.Value[:4])
			}
		}
	}
	return 0
}

func decryptPassword(encryptedPassword, secret string, authenticator []byte) string {
	if len(encryptedPassword) == 0 || len(encryptedPassword) > 128 {
		return ""
	}

	hash := md5.Sum(append([]byte(secret), authenticator...))
	decrypted := make([]byte, len(encryptedPassword))

	for i := 0; i < len(encryptedPassword); i += 16 {
		for j := 0; j < 16 && i+j < len(encryptedPassword); j++ {
			decrypted[i+j] = encryptedPassword[i+j] ^ hash[j]
		}
		if i+16 < len(encryptedPassword) {
			nextInput := append([]byte(secret), encryptedPassword[i:i+16]...)
			hash = md5.Sum(nextInput)
		}
	}

	end := len(decrypted)
	for end > 0 && decrypted[end-1] == 0 {
		end--
	}

	return string(decrypted[:end])
}

func (s *Server) buildResponse(code byte, identifier byte, authenticator [16]byte) []byte {
	buf := make([]byte, 20)
	buf[0] = code
	buf[1] = identifier
	binary.BigEndian.PutUint16(buf[2:4], 20)

	hash := md5.New()
	hash.Write(buf[:4])
	hash.Write(make([]byte, 16))
	hash.Write(authenticator[:])
	hash.Write([]byte(s.secret))
	copy(buf[4:20], hash.Sum(nil))

	return buf
}

func (s *Server) sendAccept(request *RadiusPacket, conn *net.UDPConn, addr *net.UDPAddr) {
	response := s.buildResponse(AccessAccept, request.Identifier, request.Authenticator)
	conn.WriteToUDP(response, addr)
}

func (s *Server) sendReject(request *RadiusPacket, conn *net.UDPConn, addr *net.UDPAddr) {
	response := s.buildResponse(AccessReject, request.Identifier, request.Authenticator)
	conn.WriteToUDP(response, addr)
}

func (s *Server) sendAccountingResponse(request *RadiusPacket, conn *net.UDPConn, addr *net.UDPAddr) {
	response := s.buildResponse(AccountingResponse, request.Identifier, request.Authenticator)
	conn.WriteToUDP(response, addr)
}
