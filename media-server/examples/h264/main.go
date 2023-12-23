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
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
	webrtc "github.com/pion/webrtc/v3"

	_ "github.com/pion/mediadevices/pkg/driver/screen"

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

	encoder mediadevices.MediaStream
}

func (e *ExampleEncoder) init() error {
	// vp9, err := vpx.NewVP9Params()
	// if err != nil {
	// 	return err
	// }
	// vp9.LagInFrames = 0
	// e.Metadata = vp9.RTPCodec()

	// encoder, err := vp9.BuildVideoEncoder(video.ReaderFunc(e.videoBuilder), prop.Media{Video: e.videoSetting()})
	// if err != nil {
	// 	return err
	// }
	x264Params, err := x264.NewParams()
	if err != nil {
		log.Println("x264params error", err)
		return err
	}
	x264Params.Preset = x264.PresetUltrafast
	x264Params.BitRate = 1_000_000
	e.Metadata = x264Params.RTPCodec()

	codecSelector := mediadevices.NewCodecSelector(mediadevices.WithVideoEncoders(&x264Params))

	encoder, err := mediadevices.GetDisplayMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.FrameFormat = prop.FrameFormat(frame.FormatI420)
			c.Width = prop.Int(640)
			c.Height = prop.Int(480)
		},
		Codec: codecSelector,
	})
	if err != nil {
		log.Println("encoder error get user media", err)
		return err
	}

	e.encoder = encoder

	return nil
}

func (e *ExampleEncoder) videoBuilder() (image.Image, func(), error) {
	return image.NewYCbCr(
		image.Rect(0, 0, e.Width, e.Height),
		image.YCbCrSubsampleRatio420,
	), func() {}, nil
}

func (e *ExampleEncoder) videoSetting() prop.Video {
	return prop.Video{
		Width:       e.Width,
		Height:      e.Height,
		FrameRate:   1,
		FrameFormat: frame.FormatI420,
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

	encoder, err := NewExampleEncoder(640, 480)
	if err != nil {
		log.Println(err)
		return
	}

	track, err := webrtc.NewTrackLocalStaticRTP(encoder.Metadata.RTPCodecCapability, "vp9", "local-vp9")
	if err != nil {
		log.Println(err)
		return
	}

	_, err = peerConnection.AddTrack(track)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		videoTrack := encoder.encoder.GetVideoTracks()[0]

		rtpReader, _ := videoTrack.NewRTPReader(encoder.Metadata.MimeType, rand.Uint32(), mtu)

		buf := make([]byte, mtu)
		for {
			pkts, release, err := rtpReader.Read()
			if err != nil {
				log.Println("Rtp generator error", err)
			}

			for _, pkt := range pkts {
				n, err := pkt.MarshalTo(buf)
				if err != nil {
					log.Println("rtp packet error", err)
				}

				_, err = track.Write(buf[:n])

			}
			release()
		}
	}()

	// go func() {
	// 	for {
	// 		frame, err := encoder.Read()
	// 		if err != nil {
	// 			log.Println("Error", err)
	// 		}
	//
	// 		track.Write(frame)
	//
	// 		log.Println(frame)
	// 		// time.Sleep(time.Second * 5)
	// 	}
	// }()

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
