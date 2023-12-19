package peercontext

import (
	"context"
	"fmt"
	"log/slog"

	webrtc "github.com/pion/webrtc/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
)

type peerContext struct {
	api            *webrtc.API
	logger         *slog.Logger
	peerConnection *webrtc.PeerConnection
	peerID         string

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (p *peerContext) Info() *protocol.PeerInfo {
	return &protocol.PeerInfo{
		ID:   p.peerID,
		Name: p.peerID,
	}
}

func (p *peerContext) OnDataChannel() {
	p.peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open.", d.Label(), d.ID())
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})
}

// The offer must contain at least 1 ice-ufrag. If the offer does not contain media, it return an error.
// pion error: `webrtc.ErrSessionDescriptionMissingIceUfrag`
//
// m=application 9 UDP/DTLS/SCTP webrtc-datachannel
// ...
// a=ice-ufrag:pWPIeSRyibdzXpco
func (p *peerContext) SetRemoteSessionDescriptor(offer string) error {
	return p.peerConnection.SetRemoteDescription(
		webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  offer,
		},
	)
}

func (p *peerContext) GenerateSDPAnswer() (string, error) {
	answerSessionDescriptor, err := p.peerConnection.CreateAnswer(&webrtc.AnswerOptions{})
	if err != nil {
		return "", err
	}

	if err := p.peerConnection.SetLocalDescription(answerSessionDescriptor); err != nil {
		return "", err
	}

	// The main part needs to gather ICE candidates. If it doesn't gather them, the remote peer will be unable to connect.
	<-webrtc.GatheringCompletePromise(p.peerConnection)

	return p.peerConnection.LocalDescription().SDP, nil
}

func (p *peerContext) Cancel(reason error) {
	if err := p.peerConnection.Close(); err != nil {
	}
	p.cancel(reason)
}

type NewPeerContext_Params struct {
	API           *webrtc.API
	Logger        *slog.Logger
	ParentContext context.Context
	PeerID        string
}

func NewPeerContext(params NewPeerContext_Params) (*peerContext, error) {
	ctx, cancel := context.WithCancelCause(params.ParentContext)

	peerConnection, err := params.API.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}

	return &peerContext{
		peerConnection: peerConnection,
		api:            params.API,
		logger:         params.Logger,
		peerID:         params.PeerID,
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}
