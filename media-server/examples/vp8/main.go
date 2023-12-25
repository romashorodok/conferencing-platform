package main

// https://stackoverflow.com/questions/68859120/how-to-convert-vp8-interframe-into-image-with-pion-webrtc
// https://en.wikipedia.org/wiki/Inter_frame

// Example how to make encoder in golang
// https://github.com/pion/mediadevices/blob/6829d71e588f7a5430f3d9850426bd612bf120b3/pkg/codec/vpx/vpx.go#L5

// Or like here
// https://github.com/pion/mediadevices/blob/6829d71e588f7a5430f3d9850426bd612bf120b3/pkg/codec/vaapi/vp9.go#L6

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/rtp"
	webrtc "github.com/pion/webrtc/v3"

	"github.com/romashorodok/conferencing-platform/media-server/pkg/variables"
	"github.com/romashorodok/conferencing-platform/pkg/controller/ingress"
)

var OFFER_TYPE = "offer"

type ExampleEncoder struct {
	Max   uint32
	State uint32

	Width    int
	Height   int
	Metadata *codec.RTPCodec

	encoder codec.ReadCloser
}

func (e *ExampleEncoder) Read() ([]byte, error) {
	bytes, flush, err := e.encoder.Read()
	if err != nil {
		return nil, err
	}
	flush()
	return bytes, nil
}

func (e *ExampleEncoder) init() error {
	vp9, err := vpx.NewVP9Params()
	if err != nil {
		return err
	}
	vp9.LagInFrames = 0
	vp9.BitRate = 1_000_000
	e.Metadata = vp9.RTPCodec()

	encoder, err := vp9.BuildVideoEncoder(e.videoReader(), prop.Media{Video: e.videoSetting()})
	if err != nil {
		return err
	}

	e.encoder = encoder

	return nil
}

func (e *ExampleEncoder) videoReader() video.Reader {
	var frameRate float32 = 30
	tick := time.NewTicker(time.Duration(float32(time.Second) / frameRate))
	return video.ReaderFunc(func() (img image.Image, release func(), err error) {
		i := image.RGBA{
			Pix:    make([]uint8, e.Width*e.Height*4), // 4 bytes per pixel (RGBA)
			Stride: e.Width * 4,
			Rect:   image.Rect(0, 0, e.Width, e.Height),
		}

		// Fill the entire image with a color
		fillColor := color.RGBA{255, 0, 0, 255}
		for idx := 0; idx < len(i.Pix); idx += 4 {
			i.Pix[idx] = fillColor.R
			i.Pix[idx+1] = fillColor.G
			i.Pix[idx+2] = fillColor.B
			i.Pix[idx+3] = fillColor.A
		}

		<-tick.C

		return &i, func() {}, nil
	})
}

func (e *ExampleEncoder) videoSetting() prop.Video {
	return prop.Video{
		Width:     e.Width,
		Height:    e.Height,
		FrameRate: 30,
	}
}

func NewExampleEncoder(width, height int) (*ExampleEncoder, error) {
	encoder := &ExampleEncoder{
		Max:    3,
		Width:  width,
		Height: height,
	}

	if err := encoder.init(); err != nil {
		return nil, err
	}

	return encoder, nil
}

func SendOffer(host string, offer string) (string, error) {
	payload := ingress.WebrtcHttpIngestRequest{
		Offer: &ingress.SessionDescription{
			Sdp:  &offer,
			Type: &OFFER_TYPE,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", errors.Join(errors.New("unable marshal"), err)
	}

	resp, err := http.Post(host, "application/sdp", bytes.NewReader(body))
	if err != nil {
		return "", errors.Join(errors.New("unable send request."), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		message, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", errors.New("Something so wrong even cannot read response")
		}
		return "", errors.New(string(message))
	}

	var result ingress.WebrtcHttpIngestResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", errors.Join(errors.New("json decoder error."), err)
	}

	return *result.Answer.Sdp, nil
}

const mtu = 1000

type RTPReadCloser interface {
	Read() (pkts []*rtp.Packet, release func(), err error)
	Close() error
	codec.Controllable
}

type rtpReadCloserImpl struct {
	readFn       func() ([]*rtp.Packet, func(), error)
	closeFn      func() error
	controllerFn func() codec.EncoderController
}

func (r *rtpReadCloserImpl) Read() ([]*rtp.Packet, func(), error) {
	return r.readFn()
}

func (r *rtpReadCloserImpl) Close() error {
	return r.closeFn()
}

func (r *rtpReadCloserImpl) Controller() codec.EncoderController {
	return r.controllerFn()
}

type samplerFunc func() uint32

// newVideoSampler creates a video sampler that uses the actual video frame rate and
// the codec's clock rate to come up with a duration for each sample.
func newVideoSampler(clockRate uint32) samplerFunc {
	clockRateFloat := float64(clockRate)
	lastTimestamp := time.Now()

	return samplerFunc(func() uint32 {
		now := time.Now()
		duration := now.Sub(lastTimestamp).Seconds()
		samples := uint32(math.Round(clockRateFloat * duration))
		lastTimestamp = now
		return samples
	})
}

// newAudioSampler creates a audio sampler that uses a fixed latency and
// the codec's clock rate to come up with a duration for each sample.
func newAudioSampler(clockRate uint32, latency time.Duration) samplerFunc {
	samples := uint32(math.Round(float64(clockRate) * latency.Seconds()))
	return samplerFunc(func() uint32 {
		return samples
	})
}

func main() {
	var mediaServerUri string
	flag.StringVar(&mediaServerUri, "media-server", fmt.Sprintf("http://localhost:%s/ingress/whip/session/test", variables.HTTP_PORT_DEFAULT), "host and port for media server")
	flag.Parse()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Println("peerConnection error.", err)
		return
	}
	defer peerConnection.Close()
	dataChannel, _ := peerConnection.CreateDataChannel("signaling", nil)

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
	})

	encoder, err := NewExampleEncoder(640, 480)
	if err != nil {
		log.Println(err)
		return
	}

	track, err := webrtc.NewTrackLocalStaticRTP(encoder.Metadata.RTPCodecCapability, "vp8", "local-vp8")
	if err != nil {
		log.Println(err)
		return
	}

	_, err = peerConnection.AddTrack(track)
	if err != nil {
		log.Println(err)
	}

	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rtpme/2a8acef0-4fd2-4243-a8de-45254c0b901a#gt_bc2beb5b-66f9-435f-9ffd-f7cf713f836a
	ssrc := rand.Uint32()

	packetizer := rtp.NewPacketizer(
		mtu,
		uint8(encoder.Metadata.PayloadType),
		ssrc,
		encoder.Metadata.Payloader,
		rtp.NewRandomSequencer(),
		encoder.Metadata.ClockRate,
	)
	_ = packetizer

	sampler := newVideoSampler(encoder.Metadata.ClockRate)

	rtpReaderCloser := &rtpReadCloserImpl{
		readFn: func() ([]*rtp.Packet, func(), error) {
			encoded, err := encoder.Read()
			if err != nil {
				return nil, func() {}, err
			}
			pkts := packetizer.Packetize(encoded, sampler())
			return pkts, nil, nil
		},
		closeFn: func() error {
			return nil
		},
		controllerFn: func() codec.EncoderController {
			return nil
		},
	}

	go func() {
		buf := make([]byte, mtu)
		for {
			pkts, _, _ := rtpReaderCloser.Read()

			for _, pkt := range pkts {
				n, err := pkt.MarshalTo(buf)
				if err != nil {
					log.Println("rtp packet error", err)
				}
				_, _ = track.Write(buf[:n])
			}

			// frame, err := encoder.Read()
			// if err != nil {
			// 	log.Println("Error", err)
			// }

			// packetizer := rtp.NewPacketizer(uint16(mtu), uint8(selectedCodec.PayloadType), ssrc, encoder.Metadata.Payloader, rtp.NewRandomSequencer(), selectedCodec.ClockRate)
			//
			// return &rtpReadCloserImpl{
			// 	readFn: func() ([]*rtp.Packet, func(), error) {
			// 		encoded, release, err := encodedReader.Read()
			// 		if err != nil {
			// 			encodedReader.Close()
			// 			track.onError(err)
			// 			return nil, func() {}, err
			// 		}
			// 		defer release()
			//
			// 		pkts := packetizer.Packetize(encoded.Data, encoded.Samples)
			// 		return pkts, release, err
			// 	},
			// 	closeFn:      encodedReader.Close,
			// 	controllerFn: encodedReader.Controller,
			// }, nil

			// pkt := codecs.VP9Packet{}
			// bytes, err := pkt.Unmarshal(frame)
			// if err != nil {
			// 	log.Println("Failed make rtp vp9 packet")
			// }
			// track.Write(bytes)
			// log.Println(bytes)

			time.Sleep(time.Second / 30)
			// track.Write(frame)
			// log.Println(frame)

			// time.Sleep(time.Second * 5)
		}
	}()

	peerConnection.OnICEConnectionStateChange(
		func(connectionState webrtc.ICEConnectionState) {
			log.Printf(
				"PeerConnection State has changed %s \n",
				connectionState.String(),
			)
		},
	)

	peerConnection.OnICECandidate(
		func(candidate *webrtc.ICECandidate) {
			if candidate != nil {
				log.Printf("PeerConnection ice candidate", candidate.String)
				log.Printf("PeerConnection ice candidate", candidate.Port)
				log.Printf("PeerConnection ice candidate", candidate.Protocol)
				log.Printf("PeerConnection ice candidate", candidate.Address)
			}
		},
	)

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Println("create offer error.", err)
		return
	}

	peerConnection.SetLocalDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offer.SDP,
	})

	answer, err := SendOffer(mediaServerUri, offer.SDP)
	if err != nil {
		log.Println("send offer error.", err)
		return
	}

	peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answer,
	})

	log.Println("\n", answer)
	log.Println("Sned offer:\n", offer.SDP)

	<-webrtc.GatheringCompletePromise(peerConnection)
	<-interrupt
}
