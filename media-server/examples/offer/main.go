package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/variables"
	"github.com/romashorodok/conferencing-platform/pkg/controller/ingress"
)

var OFFER_TYPE = "offer"

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

	var result ingress.WebrtcHttpIngestResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", errors.Join(errors.New("json decoder error."), err)
	}

	return *result.Answer.Sdp, nil
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
	dataChannel, _ := peerConnection.CreateDataChannel("datachann-ugrag", nil)
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())
		for range time.NewTicker(5 * time.Second).C {
			sendTextErr := dataChannel.SendText(uuid.New().String())
			if sendTextErr != nil {
				panic(sendTextErr)
			}
		}
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
	})

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
