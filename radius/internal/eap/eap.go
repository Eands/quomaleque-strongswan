package eap

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"

	"layeh.com/radius"
	"layeh.com/radius/rfc2759"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
)

// AuthMethod identifies the authentication mechanism in a RADIUS request.
type AuthMethod int

const (
	AuthMethodUnknown  AuthMethod = iota
	AuthMethodMSCHAPv2            // EAP-MSCHAPv2 via EAP-Message attribute (type 79)
	AuthMethodPAP                 // PAP via User-Password attribute (type 2)
)

const (
	eapCodeRequest  = 1
	eapCodeResponse = 2

	eapTypeIdentity = 1
	mschapv2Type    = 26

	opCodeChallenge = 1
	opCodeResponse  = 2
	opCodeSuccess   = 3
	opCodeFailure   = 4

	mschapv2IDMin = 1
	mschapv2IDMax = 255
)

// State tracks EAP conversation between Identity and MSCHAPv2 phases.
type State struct {
	Username               string
	AuthenticatorChallenge []byte
}

var (
	sessions sync.Map // map[stateToken] *State
)

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// DetectMethod checks which authentication method is used in the request.
func DetectMethod(r *radius.Request) AuthMethod {
	if len(rfc2869.EAPMessage_Get(r.Packet)) > 0 {
		return AuthMethodMSCHAPv2
	}
	if _, ok := r.Packet.Attributes.Lookup(rfc2865.UserPassword_Type); ok {
		return AuthMethodPAP
	}
	return AuthMethodUnknown
}

// HandleIdentity processes an EAP-Response/Identity and returns an Access-Challenge
// with EAP-Request/MSCHAPv2-Challenge. The state is stored for later verification.
func HandleIdentity(r *radius.Request, username string) (*radius.Packet, string, error) {
	challenge := make([]byte, 16)
	if _, err := rand.Read(challenge); err != nil {
		return nil, "", fmt.Errorf("generate challenge: %w", err)
	}

	token := generateToken()
	sessions.Store(token, &State{
		Username:               username,
		AuthenticatorChallenge: challenge,
	})

	resp := r.Response(radius.CodeAccessChallenge)
	eapReq := buildMSCHAPv2Challenge(r.Packet.Identifier+1, challenge, "strongSwan")
	rfc2869.EAPMessage_Set(resp, eapReq)

	return resp, token, nil
}

// HandleResponse processes an EAP-Response/MSCHAPv2-Response. It retrieves session
// state by token, verifies the NT response, and returns Accept or Reject.
func HandleResponse(r *radius.Request, token string, ntHashHex string) *radius.Packet {
	state, ok := sessions.LoadAndDelete(token)
	if !ok {
		return r.Response(radius.CodeAccessReject)
	}
	s := state.(*State)

	eapData := rfc2869.EAPMessage_Get(r.Packet)
	peerChallenge, ntResponse, err := parseMSCHAPv2Response(eapData)
	if err != nil {
		return r.Response(radius.CodeAccessReject)
	}

	ntHash, _ := hex.DecodeString(ntHashHex)
	if len(ntHash) != 16 {
		return r.Response(radius.CodeAccessReject)
	}

	// Compute expected NT response and compare
	expectedNT := rfc2759.ChallengeResponse(
		rfc2759.ChallengeHash(peerChallenge, s.AuthenticatorChallenge, []byte(s.Username)),
		ntHash,
	)
	if !equal(expectedNT, ntResponse) {
		return r.Response(radius.CodeAccessReject)
	}

	return r.Response(radius.CodeAccessAccept)
}

// eapCode returns the EAP code from the raw EAP-Message bytes.
func eapCode(data []byte) byte {
	if len(data) < 1 {
		return 0
	}
	return data[0]
}

// eapType returns the EAP type from the raw EAP-Message bytes.
func eapType(data []byte) byte {
	if len(data) < 5 {
		return 0
	}
	return data[4]
}

// buildMSCHAPv2Challenge builds a raw EAP-Request/MSCHAPv2-Challenge packet.
func buildMSCHAPv2Challenge(id byte, challenge []byte, serverName string) []byte {
	msLen := 5 + len(challenge) + len(serverName) // header(5) + challenge + name
	if msLen > 255 {
		msLen = 255
	}

	buf := make([]byte, 0, 5+msLen)

	// EAP header
	buf = append(buf, eapCodeRequest, id)                              // Code, ID
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(buf)+2+msLen)) // Length

	// MSCHAPv2 challenge
	buf = append(buf, mschapv2Type, opCodeChallenge)        // Type, OpCode
	buf = append(buf, id)                                   // MS-CHAPv2-ID
	buf = binary.BigEndian.AppendUint16(buf, uint16(msLen)) // MS-Length
	buf = append(buf, byte(len(challenge)))                 // Value-Size
	buf = append(buf, challenge...)                         // Challenge
	buf = append(buf, []byte(serverName)...)                // Name

	return buf
}

// parseMSCHAPv2Response extracts peer challenge and NT response from EAP-Message.
func parseMSCHAPv2Response(data []byte) (peerChallenge, ntResponse []byte, err error) {
	if len(data) < 58 {
		return nil, nil, fmt.Errorf("MSCHAPv2 response too short: %d bytes", len(data))
	}
	// data[0:4] = EAP header
	// data[4] = Type = 26 (MSCHAPv2)
	// data[5] = OpCode = 2 (Response)
	// data[6] = MS-CHAPv2-ID
	// data[7:9] = MS-Length
	// data[9] = Value-Size
	// data[10:26] = PeerChallenge (16)
	// data[26:34] = Reserved (8)
	// data[34:58] = NTResponse (24)
	peerChallenge = make([]byte, 16)
	copy(peerChallenge, data[10:26])
	ntResponse = make([]byte, 24)
	copy(ntResponse, data[34:58])
	return
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	diff := byte(0)
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
