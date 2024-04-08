package peercontext

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	webrtc "github.com/pion/webrtc/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
)

type peerContext struct {
	api    *webrtc.API
	logger *slog.Logger

	peerID         string
	peerConnection *webrtc.PeerConnection
	// Must be used only for sending offers and recv answers
	peerSignalingChannel *webrtc.DataChannel
	sfu                  *sfu.SelectiveForwardingUnit

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (p *peerContext) GetSignalingChannel() (*webrtc.DataChannel, error) {
	if p.peerSignalingChannel == nil {
		return nil, protocol.ErrPeerSignalingChannelNotFound
	}
	log.Println("signaling function", p.peerSignalingChannel)
	return p.peerSignalingChannel, nil
}

func (p *peerContext) GetPeerConnection() *webrtc.PeerConnection {
	return p.peerConnection
}

func (p *peerContext) Info() *protocol.PeerInfo {
	return &protocol.PeerInfo{
		ID:   p.peerID,
		Name: p.peerID,
	}
}

type localPeerAnswer struct {
	Type string
	Sdp  string
}

func (p *peerContext) setupSignalingChannel(dc *webrtc.DataChannel) {
	if protocol.PeerSignalingChannelLabel != dc.Label() {
		return
	}
	p.peerSignalingChannel = dc
	log.Println("set signaling channel")

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var answer localPeerAnswer
		json.Unmarshal(msg.Data, &answer)

		desc := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  answer.Sdp,
		}

		if err := p.peerConnection.SetLocalDescription(desc); err != nil {
			log.Println("Unable set answer", err)
			return
		}
		log.Println("Success answer set")
	})
}

func (p *peerContext) OnDataChannel() {
	p.peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		// stats, ok := p.peerConnection.GetStats().GetDataChannelStats(dc)
		// if !ok {
		// 	slog.Error(fmt.Sprintf("unable get data_channel stats for %s:%s", dc.Label(), dc.ID()))
		// 	_ = dc.Close()
		// 	return
		// }
		p.setupSignalingChannel(dc)

		// logger := p.logger.With(
		// 	slog.Group("data_channel",
		// 		slog.String("stats.ID:", stats.ID),
		//
		// 		slog.Int("dc.ID", int(*dc.ID())),
		// 		slog.String("dc.label", dc.Label()),
		// 	),
		// )

		// logger.Debug("OnDataChannel connect.")

		// dc.OnOpen(func() {
		// 	logger.Debug("OnDataChannel OnOpen")
		// })
		//
		// dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		// 	logger.Debug(fmt.Sprintf("OnDataChannel OnMessage: %s", msg.Data))
		// })
	})
}

func (p *peerContext) OnTrack(signaling func()) {
	p.peerConnection.OnTrack(func(track *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
		localTrack, err := p.sfu.AddTrack(track)
		if err != nil {
			p.logger.Error(fmt.Sprint("Failed add track to the sfu", err))
			return
		}
		defer p.sfu.RemoveTrack(localTrack)
		signaling()

		for {
			rtp, _, err := track.ReadRTP()
			if err != nil {
				log.Println("read error", err)
				return
			}

			if err = localTrack.WriteRTP(rtp); err != nil {
				return
			}
		}
	})
}

func (p *peerContext) OnCandidate() {
	p.peerConnection.OnICECandidate(
		func(candidate *webrtc.ICECandidate) {
			if candidate != nil {
				p.logger.Debug(fmt.Sprintf("On ICE candidate: %+v", candidate))
			}
		},
	)
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

var _ protocol.PeerContext = (*peerContext)(nil)

type NewPeerContext_Params struct {
	API           *webrtc.API
	Logger        *slog.Logger
	SFU           *sfu.SelectiveForwardingUnit
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
		api:            params.API,
		logger:         params.Logger,
		peerConnection: peerConnection,
		sfu:            params.SFU,
		peerID:         params.PeerID,
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}
