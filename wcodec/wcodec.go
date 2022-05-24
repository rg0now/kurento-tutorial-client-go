package wcodec

import (
	"os"
	"fmt"
	"log"
	"context"
	"time"
	"io"
	"net"
	"strconv"
	"strings"
	
	"github.com/pion/webrtc/v3"
	"github.com/pion/rtp"
	"github.com/pion/rtcp"
	"github.com/pion/sdp/v3"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
	// "github.com/pion/webrtc/v3/pkg/media/h264writer"
)
	
// codec defs: from RegisterDefaultCodecs
const (
	// oggPageDuration   = time.Millisecond * 20
	h264FrameDuration = time.Millisecond * 33
)

var videoRTCPFeedback = []webrtc.RTCPFeedback{{"goog-remb", ""}, {"ccm", "fir"}, {"nack", ""}, {"nack", "pli"}}

var VP8Codecs = []webrtc.RTPCodecParameters {
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeVP8, 90000, 0, "", videoRTCPFeedback},
		PayloadType:        96,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=96", nil},
		PayloadType:        97,
	},
}

var H264Codecs = []webrtc.RTPCodecParameters {
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0,
			"level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", videoRTCPFeedback},
		PayloadType:        102,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=102", nil},
		PayloadType:        121,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0,
			"level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", videoRTCPFeedback},
		PayloadType:        127,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=127", nil},
		PayloadType:        120,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0,
			"level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", videoRTCPFeedback},
		PayloadType:        125,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=125", nil},
		PayloadType:        107,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0,
			"level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f", videoRTCPFeedback},
		PayloadType:        108,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=108", nil},
		PayloadType:        109,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0,
			"level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", videoRTCPFeedback},
		PayloadType:        127,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=127", nil},
		PayloadType:        120,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0,
			"level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640032", videoRTCPFeedback},
		PayloadType:        123,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=123", nil},
		PayloadType:        118,
	},
}

// transmitters: disk -> WebRTC
func SendFile(ctx context.Context, rtpSender *webrtc.RTPSender, file, codec string,
	track *webrtc.TrackLocalStaticSample) {
	
	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things like
	// NACK this needs to be called.
	go func() {
		for {
			if rtcps, _, rtcpErr := rtpSender.ReadRTCP(); rtcpErr != nil {
				for _, rtcp := range rtcps {
					log.Println(rtcp)
				}
			}
		}
	}()
	
	switch codec {
	case webrtc.MimeTypeVP8:
		go sendIvfFile(ctx, file, track)
	case webrtc.MimeTypeH264:	
		go sendH264File(ctx, file, track)
	}
}

func sendIvfFile(ctx context.Context, fileName string, track *webrtc.TrackLocalStaticSample) {
	// Open a IVF file and start reading using our IVFReader
	file, ivfErr := os.Open(fileName)
	if ivfErr != nil {
		log.Fatalln(ivfErr)
	}

	ivf, header, ivfErr := ivfreader.NewWith(file)
	if ivfErr != nil {
		log.Fatalln(ivfErr)
	}

	// Wait for connection established
	<-ctx.Done()

	// Send our video file frame at a time. Pace our sending so we send it at the same
	// speed it should be played back as.
	// This isn't required since the video is timestamped, but we will such much higher
	// loss if we send all at once.
	//
	// It is important to use a time.Ticker instead of time.Sleep because * avoids
	// accumulating skew, just calling time.Sleep didn't compensate for the time spent
	// parsing the data * works around latency issues with Sleep (see
	// https://github.com/golang/go/issues/44343)
	ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator) /
		float32(header.TimebaseDenominator))*1000))
	for ; true; <-ticker.C {
		frame, _, ivfErr := ivf.ParseNextFrame()
		if ivfErr == io.EOF {
			log.Println("End of video")
			os.Exit(0)
		}

		if ivfErr != nil {
			log.Fatalln(ivfErr)
		}

		if ivfErr = track.WriteSample(media.Sample{Data: frame,
			Duration: time.Second}); ivfErr != nil {
				log.Fatalln(ivfErr)
			}
	}
}

func sendH264File(ctx context.Context, fileName string, track *webrtc.TrackLocalStaticSample) {
	// Open a H264 file and start reading using our IVFReader
	file, h264Err := os.Open(fileName)
	if h264Err != nil {
		log.Fatalln(h264Err)
	}

	h264, h264Err := h264reader.NewReader(file)
	if h264Err != nil {
		log.Fatalln(h264Err)
	}

	// Wait for connection established
	<-ctx.Done()

	// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
	// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
	//
	// It is important to use a time.Ticker instead of time.Sleep because
	// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
	// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
	ticker := time.NewTicker(h264FrameDuration)
	for ; true; <-ticker.C {
		nal, h264Err := h264.NextNAL()
		if h264Err == io.EOF {
			log.Printf("All video frames parsed and sent")
			os.Exit(0)
		}
		if h264Err != nil {
			log.Fatalln(h264Err)
		}

		if h264Err = track.WriteSample(media.Sample{Data: nal.Data, Duration: time.Second}); h264Err != nil {
			log.Fatalln(h264Err)
		}
	}
}

// receivers: WebRTC -> disk
func ReceiveTrack(peerConnection *webrtc.PeerConnection, file, codec string) func (*webrtc.TrackRemote, *webrtc.RTPReceiver) {

	switch codec {
	case webrtc.MimeTypeVP8:
		// curry
		return func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) () {
			receiveVP8Track(track, peerConnection, file)
		}
	case webrtc.MimeTypeH264:	
		return func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			receiveH264Track(track, peerConnection, file)
		}
	}

	panic("This can never happen")
}

func receiveVP8Track(track *webrtc.TrackRemote, peerConnection *webrtc.PeerConnection, file string) {

	// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for range ticker.C {
			errSend := peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
			if errSend != nil {
				log.Println(errSend)
			}
		}
	}()

	ivfFile, err := ivfwriter.New(file)
	if err != nil {
		log.Fatalln(err)
	}
	defer ivfFile.Close()
	
	codec := track.Codec()
	if strings.EqualFold(codec.MimeType, webrtc.MimeTypeOpus) {
		log.Fatalln("Got Opus track: unimplemented")
		// saveToDisk(oggFile, track)
	} else if strings.EqualFold(codec.MimeType, webrtc.MimeTypeVP8) {
		log.Println("Got VP8 track, saving to disk as output.ivf")
		for {
			rtpPacket, _, err := track.ReadRTP()
			if err != nil {
				log.Fatalln(err)
			}
			if err := ivfFile.WriteRTP(rtpPacket); err != nil {
				log.Fatalln(err)
			}
		}
	}
}
		
func receiveH264Track(track *webrtc.TrackRemote, peerConnection *webrtc.PeerConnection, file string) {
	log.Fatalln("ReceiveH264Track: Unimplemented")
}
		
		
//////////////////////////
// transmitters: disk -> WebRTC
func createConnections(offer, answer *webrtc.SessionDescription) (*net.UDPConn, *net.UDPConn) {
	// local addr:port: first candidate
	// remote addr: answer.c=...
	// remote port: answer.m=...
	var laddr, raddr *net.UDPAddr
	var ssrc uint32
	
	// offer
	parsedOffer, err := offer.Unmarshal()
	if err != nil {
		log.Fatal("cannot parse SDP:", offer)
	}
	
	if len(parsedOffer.MediaDescriptions) > 0 {
		m := parsedOffer.MediaDescriptions[0]
		for _, a := range m.Attributes {
			if a.IsICECandidate() {
				candidateValue := strings.TrimPrefix(a.Value, "candidate:")

				c, err := ice.UnmarshalCandidate(candidateValue)
				if err != nil {
					log.Printf("cannot parse ICE candidate '%s': %s", c, err)
					continue
				}

				if laddr, err = net.ResolveUDPAddr("udp",
					fmt.Sprintf("%s:%d", c.Address(), c.Port())); err != nil {
						log.Printf("cannot parse address from ICE candidate '%s': %s", c, err)
						continue
					}
				break
			}
			if a.Key == sdp.AttrKeySSRC {
				s := strings.Split(a.Value, " ")
				if len(s) == 0 {
					log.Println("cannot split attribute:", a.Value)
					continue
				}
				u64, err := strconv.ParseUint(s[0], 10, 32)
				if err != nil {
					log.Println("cannot parse SSRC from attribute:", a.Value)
					continue
				}
				ssrc = uint32(u64)
			}
		}
	} else {
		log.Fatal("cannot find media info (m=) in SDP:", offer)
	}
	
	if laddr == nil {
		log.Fatal("no ICE candidate found in SDP:", offer)
	}

	parsedAnswer, err := answer.Unmarshal()
	if err != nil {
		log.Fatal("cannot parse SDP:", answer)
	}

	if len(parsedAnswer.MediaDescriptions) > 0 && parsedAnswer.ConnectionInformation != nil &&
		parsedAnswer.ConnectionInformation.Address != nil {
		m := parsedAnswer.MediaDescriptions[0]
		p := m.MediaName.Port.Value
		r := parsedAnswer.ConnectionInformation.Address.Address
		if raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", r, p)); err != nil {
			log.Fatalf("cannot parse address(%s):port(%d) from SDP: %s", r, p, err)
		}	
		
	} else {
		log.Fatal("cannot find media info (m=) in SDP:", answer)
	}
	
	if raddr == nil {
		log.Fatal("no media info found in SDP:", answer)
	}

	// RTP
	rtpConn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		log.Fatalf("could not open RTP connection: %s", err)
	}
	
	defer func(conn net.UDPConn) {
		if closeErr := conn.Close(); closeErr != nil {
			log.Fatalf("could not close RTP connection: %s -> %s: %s", laddr, raddr, closeErr)
		}
	}(*rtpConn)

	// RTCP: RTP port + 1
	if laddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", laddr.IP, laddr.Port+1)); err != nil {
		log.Fatalf("cannot create local RTCP address: %s", err)
	}
	if raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", raddr.IP, raddr.Port+1)); err != nil {
		log.Fatalf("cannot create remote RTCP address: %s", err)
	}	

	rtcpConn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		log.Fatalf("could not open RTCP connection: %s", err)
	}
	
	defer func(conn net.UDPConn) {
		if closeErr := conn.Close(); closeErr != nil {
			log.Fatalf("could not close RTCP connection: %s -> %s: %s", laddr, raddr, closeErr)
		}
	}(*rtcpConn)
	
	// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
	go func() {
		ticker := time.NewTicker(time.Second * 2)
		for range ticker.C {
			ps := []rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: ssrc}}
			for _, p := range ps {
				buf, err := p.Marshal()
				if err != nil {
					log.Println("cannot marshal RTCP packer:", p)
				}
				if _, err := rtcpConn.Write(buf); err != nil {
					log.Println(err)
				}
			}
		}
	}()

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things like
	// NACK this needs to be called.
	go func() {
		buf := make([]byte, 2000)
		for {
			_, _, err := rtcpConn.ReadFrom(buf[0:])
			if err != nil {
				// log.Fatalln("could not read RTCP packet:", err)
				log.Println("could not read RTCP packet:", err)
			}

			log.Println(string(buf))
		}
	}()
	
	return rtpConn, rtcpConn
}

func RTPSendFile(offer, answer *webrtc.SessionDescription, file, codec string,
	track *webrtc.TrackLocalStaticSample) {

	rtpConn, rtcpConn := createConnections(offer, answer)
	
	switch codec {
	case webrtc.MimeTypeVP8:
		go rtpSendIvfFile(rtpConn, rtcpConn, file, track)
	case webrtc.MimeTypeH264:	
		go rtpSendH264File(rtpConn, rtcpConn, file, track)
	}
}

func rtpSendIvfFile(rtpConn, rtcpConn *net.UDPConn, fileName string, track *webrtc.TrackLocalStaticSample) {
	// Open a IVF file and start reading using our IVFReader
	file, ivfErr := os.Open(fileName)
	if ivfErr != nil {
		log.Fatalln(ivfErr)
	}

	ivf, header, ivfErr := ivfreader.NewWith(file)
	if ivfErr != nil {
		log.Fatalln(ivfErr)
	}
	
	// Send our video file frame at a time. Pace our sending so we send it at the same
	// speed it should be played back as.
	// This isn't required since the video is timestamped, but we will such much higher
	// loss if we send all at once.
	//
	// It is important to use a time.Ticker instead of time.Sleep because * avoids
	// accumulating skew, just calling time.Sleep didn't compensate for the time spent
	// parsing the data * works around latency issues with Sleep (see
	// https://github.com/golang/go/issues/44343)
	ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator) /
		float32(header.TimebaseDenominator))*1000))
	for ; true; <-ticker.C {
		frame, _, ivfErr := ivf.ParseNextFrame()
		if ivfErr == io.EOF {
			log.Println("End of video")
			os.Exit(0)
		}

		if ivfErr != nil {
			log.Fatalln(ivfErr)
		}

		if _, err := rtpConn.Write(frame); err != nil {
			log.Fatalln(err)
		}
	}
}

func rtpSendH264File(rtpConn, rtcpConn *net.UDPConn, fileName string, track *webrtc.TrackLocalStaticSample) {
	// Open a H264 file and start reading using our IVFReader
	file, h264Err := os.Open(fileName)
	if h264Err != nil {
		log.Fatalln(h264Err)
	}

	h264, h264Err := h264reader.NewReader(file)
	if h264Err != nil {
		log.Fatalln(h264Err)
	}

	// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
	// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
	//
	// It is important to use a time.Ticker instead of time.Sleep because
	// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
	// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
	ticker := time.NewTicker(h264FrameDuration)
	for ; true; <-ticker.C {
		nal, h264Err := h264.NextNAL()
		if h264Err == io.EOF {
			log.Printf("All video frames parsed and sent")
			os.Exit(0)
		}
		if h264Err != nil {
			log.Fatalln(h264Err)
		}

		// m := media.Sample{Data: nal.Data, Duration: time.Second}
		// if _, h264Err = rtpConn.Write(m.Marshal()); h264Err != nil {
		// 	log.Fatalln(h264Err)
		// }
		if _, err := rtpConn.Write(nal.Data); err != nil {
			log.Fatalln("cannot write RTP packet:", err)
		}
	}
}

// receivers: WebRTC -> disk
func RTPReceiveTrack(offer, answer *webrtc.SessionDescription, codec, file string) {
	rtpConn, rtcpConn := createConnections(offer, answer)
	
	switch codec {
	case webrtc.MimeTypeVP8:
		rtpReceiveVP8Track(rtpConn, rtcpConn, file)
	case webrtc.MimeTypeH264:	
		rtpReceiveH264Track(rtpConn, rtcpConn, file)
	}

	panic("This can never happen")
}

func rtpReceiveVP8Track(rtpConn, rtcpConn *net.UDPConn, file string) {
	ivfFile, err := ivfwriter.New(file)
	if err != nil {
		log.Fatalln(err)
	}
	defer ivfFile.Close()
	
	buf := make([]byte, 2000)
	for {
		_, err := rtpConn.Read(buf)
		if err != nil {
			log.Fatalln("cannot read RTP packet:", err)
		}
		var p *rtp.Packet
		if err := p.Unmarshal(buf); err != nil {
			log.Println("could not parse received RTP packet:", err)
		}
		
		if err := ivfFile.WriteRTP(p); err != nil {
			log.Println(err)
		}
	}
}

func rtpReceiveH264Track(rtpConn, rtcpConn *net.UDPConn, file string) {
	log.Fatalln("ReceiveH264Track: Unimplemented")
}
		
		
