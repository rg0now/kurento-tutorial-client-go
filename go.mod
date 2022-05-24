module webrtc-client-go

go 1.15

require (
	github.com/gorilla/websocket v1.4.2
	github.com/pion/ice/v2 v2.2.2
	github.com/pion/rtcp v1.2.9
	github.com/pion/rtp v1.7.9
	github.com/pion/sdp/v3 v3.0.4
	github.com/pion/webrtc/v3 v3.1.5
)

// replace github.com/pion/webrtc/v3 => /export/l7mp/webrtc-client-go/webrtc

// replace github.com/pion/dtls/v2 => /export/l7mp/webrtc-client-go/webrtc/dtls
