package vici

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"webapp/internal/models"
)

const (
	etSectionStart = 1
	etSectionEnd   = 2
	etKeyValue     = 3
	etListStart    = 4
	etListItem     = 5
)

type Client struct {
	addr           string
	timeout        time.Duration
	reconnectInt   time.Duration
	conn           net.Conn
	mu             sync.Mutex
	running        bool
	activeSessions sync.Map
}

func NewClient(addr string, timeoutSec, reconnectSec int) *Client {
	return &Client{
		addr:         addr,
		timeout:      time.Duration(timeoutSec) * time.Second,
		reconnectInt: time.Duration(reconnectSec) * time.Second,
	}
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.Dial("tcp", c.addr)
	if err != nil {
		return fmt.Errorf("failed to connect to VICI at %s: %w", c.addr, err)
	}
	c.conn = conn
	log.Printf("Connected to VICI at %s", c.addr)
	return nil
}

func (c *Client) Run() {
	c.running = true
	for c.running {
		if err := c.Connect(); err != nil {
			log.Printf("VICI connection error: %v, retrying in %v", err, c.reconnectInt)
			time.Sleep(c.reconnectInt)
			continue
		}

		c.subscribeEvents()

		c.activeSessions.Range(func(_, _ interface{}) bool { return true })
		c.activeSessions = sync.Map{}

		c.refreshActiveSessions()

		if err := c.readEventLoop(); err != nil {
			log.Printf("VICI read error: %v, reconnecting...", err)
		}

		time.Sleep(c.reconnectInt)
	}
}

func (c *Client) subscribeEvents() {
	events := []string{"ike-updown", "child-updown"}
	for _, event := range events {
		msg := buildRegisterEvent(event)
		if err := c.sendRaw(msg); err != nil {
			log.Printf("Failed to register event %s: %v", event, err)
		} else {
			if _, err := c.readResponse(); err != nil {
				log.Printf("Failed to read event registration response for %s: %v", event, err)
			}
		}
	}
}

func (c *Client) refreshActiveSessions() {
	sas, err := c.ListSAs()
	if err != nil {
		log.Printf("Failed to list active SAs: %v", err)
		return
	}

	seen := make(map[string]bool)
	for _, sa := range sas {
		seen[sa.UniqueID] = true
		if _, exists := c.activeSessions.Load(sa.UniqueID); !exists {
			log.Printf("New IKE SA: %s (user: %s, remote: %s)", sa.UniqueID, sa.Username, sa.RemoteIP)
		}
		c.activeSessions.Store(sa.UniqueID, sa)
	}

	c.activeSessions.Range(func(key, _ interface{}) bool {
		if !seen[key.(string)] {
			c.activeSessions.Delete(key)
			log.Printf("Removed IKE SA: %s", key)
		}
		return true
	})
}

func (c *Client) ListSAs() ([]models.ActiveSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, errors.New("not connected to VICI")
	}

	if err := c.sendCommand("list-sas", nil); err != nil {
		return nil, err
	}

	msg, err := c.readResponse()
	if err != nil {
		return nil, err
	}

	var sessions []models.ActiveSession

	for _, section := range msg {
		if section.Name != "list-sas" {
			continue
		}
		for _, ikeSA := range section.Sections {
			if ikeSA.Name != "ike-sa" {
				continue
			}
			session := models.ActiveSession{
				UniqueID:   ikeSA.KVs["uniqueid"],
				State:      ikeSA.KVs["state"],
				IKEVersion: ikeSA.KVs["version"],
				RemoteIP:   ikeSA.KVs["remote-host"],
				RemoteID:   ikeSA.KVs["remote-id"],
				StartTime:  ikeSA.KVs["established"],
			}

			for _, eap := range ikeSA.Sections {
				if eap.Name == "remote-eap-id" {
					session.Username = eap.Value
				}
			}

			for _, childSA := range ikeSA.Sections {
				if childSA.Name == "child-sa" {
					session.VirtualIP = childSA.KVs["local-ts"]
					if bIn, ok := childSA.KVs["bytes-in"]; ok {
						parseUint64(bIn, &session.BytesIn)
					}
					if bOut, ok := childSA.KVs["bytes-out"]; ok {
						parseUint64(bOut, &session.BytesOut)
					}
				}
			}

			if session.Username == "" {
				session.Username = session.RemoteID
			}

			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (c *Client) readEventLoop() error {
	for c.running {
		c.mu.Lock()
		if c.conn == nil {
			c.mu.Unlock()
			return errors.New("connection lost")
		}

		msg, err := c.readResponseLocked()
		if err != nil {
			c.conn.Close()
			c.conn = nil
			c.mu.Unlock()
			return err
		}
		c.mu.Unlock()

		for _, section := range msg {
			switch section.Name {
			case "ike-updown", "child-updown":
				eventName := section.KVs["updown"]
				if eventName == "up" {
					log.Printf("VICI event %s: session up, refreshing", section.Name)
					c.refreshActiveSessions()
				} else if eventName == "down" {
					log.Printf("VICI event %s: session down, refreshing", section.Name)

					if section.Name == "ike-updown" {
						for _, ikeSA := range section.Sections {
							if ikeSA.Name == "ike" && ikeSA.Value != "" {
								if name, ok := ikeSA.KVs["name"]; ok && name != "" {
									log.Printf("Removing IKE SA: %s", name)
									c.activeSessions.Delete(ikeSA.KVs["uniqueid"])
								}
							}
						}
					}
					c.refreshActiveSessions()
				}
			}
		}
	}
	return nil
}

func (c *Client) GetActiveSessions() []models.ActiveSession {
	var sessions []models.ActiveSession
	c.activeSessions.Range(func(key, value interface{}) bool {
		if sa, ok := value.(models.ActiveSession); ok {
			sessions = append(sessions, sa)
		}
		return true
	})
	return sessions
}

func (c *Client) Close() {
	c.running = false
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) sendCommand(name string, msg []byte) error {
	return c.sendRaw(buildCommand(name, msg))
}

func (c *Client) sendRaw(data []byte) error {
	if c.conn == nil {
		return errors.New("not connected")
	}
	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err := c.conn.Write(data)
	return err
}

func (c *Client) readResponse() (ViciMessage, error) {
	return c.readResponseLocked()
}

func (c *Client) readResponseLocked() (ViciMessage, error) {
	if c.conn == nil {
		return nil, errors.New("not connected")
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)

	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return nil, err
	}

	return parse(data)
}

func buildCommand(name string, msg []byte) []byte {
	nameBytes := []byte(name)
	totalLen := 4 + len(nameBytes) + 4 + len(msg)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(totalLen))
	binary.Write(buf, binary.BigEndian, uint32(len(nameBytes)))
	buf.Write(nameBytes)
	binary.Write(buf, binary.BigEndian, uint32(len(msg)))
	buf.Write(msg)
	return buf.Bytes()
}

func buildRegisterEvent(eventName string) []byte {
	var msgBuf bytes.Buffer
	msgBuf.WriteByte(etSectionStart)
	writeStringField(&msgBuf, "event")
	writeStringField(&msgBuf, eventName)
	msgBuf.WriteByte(etSectionEnd)

	return buildCommand("register-event", msgBuf.Bytes())
}

type ViciSection struct {
	Name     string
	Value    string
	KVs      map[string]string
	Sections []*ViciSection
}

type ViciMessage []*ViciSection

func parse(data []byte) (ViciMessage, error) {
	if len(data) < 4 {
		return nil, nil
	}

	nameLen := binary.BigEndian.Uint32(data[:4])
	if len(data) < int(4+nameLen+4) {
		return nil, errors.New("invalid VICI response")
	}

	msgData := data[4+nameLen+4:]

	reader := bytes.NewReader(msgData)
	sections, _, err := parseElements(reader, 0)
	if err != nil {
		return nil, err
	}

	return sections, nil
}

func parseElements(reader *bytes.Reader, terminator byte) ([]*ViciSection, bool, error) {
	var sections []*ViciSection

	for reader.Len() > 0 {
		et, err := reader.ReadByte()
		if err != nil {
			return nil, false, err
		}

		if terminator != 0 && et == terminator {
			return sections, true, nil
		}

		switch et {
		case etSectionEnd:
			return sections, true, nil

		case etSectionStart:
			name, value, err := readNameValue(reader)
			if err != nil {
				return nil, false, err
			}
			sec := &ViciSection{
				Name:  name,
				Value: value,
				KVs:   make(map[string]string),
			}
			subSections, _, err := parseElements(reader, etSectionEnd)
			if err != nil {
				return nil, false, err
			}
			for _, ss := range subSections {
				if ss.Name == "" {
					for k, v := range ss.KVs {
						sec.KVs[k] = v
					}
				} else {
					sec.Sections = append(sec.Sections, ss)
				}
			}
			sections = append(sections, sec)

		case etKeyValue:
			key, value, err := readKeyValue(reader)
			if err != nil {
				return nil, false, err
			}
			sec := &ViciSection{KVs: map[string]string{key: value}}
			sections = append(sections, sec)

		case etListStart:
			name, _, err := readNameValue(reader)
			if err != nil {
				return nil, false, err
			}
			sec := &ViciSection{
				Name: name,
				KVs:  make(map[string]string),
			}
			subSections, _, err := parseElements(reader, etListItem)
			if err != nil {
				return nil, false, err
			}
			for _, ss := range subSections {
				if ss.Name == "" {
					for k, v := range ss.KVs {
						sec.KVs[k] = v
					}
				} else {
					sec.Sections = append(sec.Sections, ss)
				}
			}
			sections = append(sections, sec)

		case etListItem:
			return sections, true, nil
		}
	}

	return sections, false, nil
}

func readNameValue(reader *bytes.Reader) (string, string, error) {
	name, err := readStringField(reader)
	if err != nil {
		return "", "", err
	}
	value, err := readStringField(reader)
	if err != nil {
		return "", "", err
	}
	return name, value, nil
}

func readKeyValue(reader *bytes.Reader) (string, string, error) {
	key, err := readStringField(reader)
	if err != nil {
		return "", "", err
	}
	value, err := readStringField(reader)
	if err != nil {
		return "", "", err
	}
	return key, value, nil
}

func readStringField(reader *bytes.Reader) (string, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint16(buf)
	if length == 0 {
		return "", nil
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return "", err
	}
	return string(data), nil
}

func writeStringField(w io.Writer, s string) {
	binary.Write(w, binary.BigEndian, uint16(len(s)))
	w.Write([]byte(s))
}

func parseUint64(s string, dest *uint64) {
	var v uint64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + uint64(c-'0')
		}
	}
	*dest = v
}
