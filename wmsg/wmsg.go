package wmsg

import (
	"errors"
	// "fmt"

	// "github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v3"
)
	
type Message interface {
	Message()
}

// register
type RegisterRequest struct {
	Id string   `json:"id"`
	Name string `json:"name"`
}

func (RegisterRequest) Message() { return }

func NewRegisterRequest(name string) Message {
	return RegisterRequest{"register", name}
}

type RegisterResponse struct {
	Id string       `json:"id"`
	Response string `json:"response"`
}

func (RegisterResponse) Message() { return }

func NewRegisterResponse(m map[string]interface{}) (RegisterResponse, error) {
	ret := RegisterResponse{Id: m["id"].(string), Response: m["response"].(string)}
	if m["id"].(string) != "registerResponse" {
		return ret, errors.New("expected message: registerResponse")
	}
	return ret, nil
}

// call
type CallRequest struct {
	Id string   `json:"id"`
	From string `json:"from"`
	To string   `json:"to"`
	Sdp string  `json:"sdpOffer"`
}

func (CallRequest) Message() { return }

func NewCallRequest(from, to, sdp string) Message {
	return CallRequest{"call", from, to, sdp}
}

type CallResponse struct {
	Id string	`json:"id"`
	Response string `json:"response"`
	Sdp string	`json:"sdpAnswer"`
}

func (CallResponse) Message() { return }

func NewCallResponse(m map[string]interface{}) (CallResponse, error) {
	ret := CallResponse{Id: m["id"].(string), Response: m["response"].(string),
		Sdp: m["sdpAnswer"].(string)}
	if m["id"].(string) != "callResponse" {
		return ret, errors.New("expected message: callResponse")
	}
	return ret, nil
}

type IncomingCallRequest struct {
	Id string   `json:"id"`
	From string `json:"from"`
}

func (IncomingCallRequest) Message() { return }

func NewIncomingCallRequest(m map[string]interface{}) (IncomingCallRequest, error) {
	ret := IncomingCallRequest{Id: m["id"].(string), From: m["from"].(string)}
	if m["id"].(string) != "incomingCall" {
		return ret, errors.New("expected message: incomingCall")
	}
	return ret, nil
}

type IncomingCallResponse struct {
	Id string		`json:"id"`
	From string		`json:"from"`
	Response string	        `json:"callResponse"`
	Sdp string		`json:"sdpOffer"`
}

func (IncomingCallResponse) Message() { return }

func NewIncomingCallResponse(from, response, sdp string) Message {
	return IncomingCallResponse{"incomingCallResponse", from, response, sdp}
}

type StartCommunication struct {
	Id string   `json:"id"`
	Sdp string  `json:"sdpAnswer"`
}

func (StartCommunication) Message() { return }

func NewStartCommunication(m map[string]interface{}) (StartCommunication, error) {
	ret := StartCommunication{Id: m["id"].(string), Sdp: m["sdpAnswer"].(string)}
	if m["id"].(string) != "startCommunication" {
		return ret, errors.New("expected message: startCommunication")
	}
	return ret, nil
}

// --- Magic Mirror example related structures
type MagicMirrorRequest struct {
	Id string   `json:"id"`
	Sdp string  `json:"sdpOffer"`
}

func (MagicMirrorRequest) Message() { return }

func NewMagicMirrorRequest(sdp string) Message {
	return MagicMirrorRequest{"start", sdp}
}

type MagicMirrorResponse struct {
	Id string	`json:"id"`
	Sdp string	`json:"sdpAnswer"`
}

func (MagicMirrorResponse) Message() { return }

func NewMagicMirrorResponse(m map[string]interface{}) (MagicMirrorResponse, error) {
	ret := MagicMirrorResponse{Id: m["id"].(string),
		Sdp: m["sdpAnswer"].(string)}
	if m["id"].(string) != "startResponse" {
		return ret, errors.New("expected message: startResponse")
	}
	return ret, nil
}
// --------------

// ICE
type ICECandidate struct {
	Id string				`json:"id"`
	Candidate map[string]interface{}	`json:"candidate"`
}

func (ICECandidate) Message() { return }

func NewICECandidate(m map[string]interface{}) (ICECandidate, error) {
	ret := ICECandidate{Id: m["id"].(string), Candidate: m["candidate"].(map[string]interface{})}
	if m["id"].(string) != "iceCandidate" {
		return ret, errors.New("expected message: iceCandidate")
	}
	return ret, nil
}

type OnICECandidate struct {
	Candidate webrtc.ICECandidateInit  `json:"candidate"`
	Id string		           `json:"id"`
}

func (OnICECandidate) Message() { return }

func NewOnICECandidate(candidate *webrtc.ICECandidate) Message {
	init := candidate.ToJSON()
	return OnICECandidate{
		Candidate: init,
		Id: "onIceCandidate",
	}
}

////////////////
// utils
func ParseSdp(sdpType webrtc.SDPType, sdp string) (*webrtc.SessionDescription, error) {
	desc := &webrtc.SessionDescription{Type: sdpType, SDP: sdp}

	// Kurento compat hack: remove duplicate fingerprints
	sdpParsed, err := desc.Unmarshal()
	if err != nil {
		return nil, err
	}

	// fmt.Printf("before parse: %s\n", *sdpParsed)
	
	attrs := sdpParsed.Attributes
	for i, a := range  attrs{
		if a.Key == "fingerprint" {
			attrs = append(attrs[:i], attrs[i+1:]...)
			break
		}
	}
	_ = copy(sdpParsed.Attributes, attrs)
	
	// fmt.Printf("after parse: %s\n", *sdpParsed)

	// parse SDP back
	sdpB, errB := sdpParsed.Marshal()
	if errB != nil {
		return nil, errB
	}
	desc.SDP = string(sdpB)
	
	return desc, nil
}
