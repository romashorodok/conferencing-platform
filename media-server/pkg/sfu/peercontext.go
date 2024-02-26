package sfu

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"

	"github.com/google/uuid"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/rtpstats"
	"github.com/romashorodok/conferencing-platform/pkg/executils"
)

type PeerContext struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	PeerID         protocol.PeerID
	webrtc         *webrtc.API
	peerConnection *webrtc.PeerConnection
	stats          *rtpstats.RtpStats
	Signal         *Signal
	Subscriber     *Subscriber
}

type trackWritable interface {
	WriteRTP(p *rtp.Packet) error
	OnFeedback(func(pkts []rtcp.Packet))
	OnCloseAsync(func())
}

func (p *PeerContext) OnTrack(pool *PeerContextPool) {
	p.peerConnection.OnTrack(func(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
		defer func() {
			for _, peer := range pool.Get() {
				_ = peer.Subscriber.DeleteTrack(t.ID())
			}
		}()
		log.Println("On track", t.ID())

		var threshold uint64 = 1000000
		var step uint64 = 2

		for {
			select {
			case <-p.ctx.Done():
				return
			default:
			}

			pkt, _, err := t.ReadRTP()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				continue
			}

			executils.ParallelExec(pool.Get(), threshold, step, func(peer *PeerContext) {
				select {
				case <-peer.ctx.Done():
					return
				default:
				}

				var track trackWritable
				var exist bool
				var err error

				switch {
				case p.PeerID == peer.PeerID:
					track, exist = peer.Subscriber.HasLoopbackTrack(t.ID())
					if !exist {
						log.Println("Create loopback track")
						track, err = peer.Subscriber.LoopbackTrack(t, recv)
						track.OnFeedback(func(pkts []rtcp.Packet) {
							if err := peer.peerConnection.WriteRTCP(pkts); err != nil {
								log.Printf("transport-cc ERROR | %s", err)
							}
						})
						track.OnCloseAsync(func() {
							_ = peer.Signal.DispatchOffer()
						})
					}

				default:
					track, exist = peer.Subscriber.HasTrack(t.ID())
					if !exist {
						log.Println("Create local track track")
						track, err = peer.Subscriber.Track(t, recv)
						track.OnFeedback(func(pkts []rtcp.Packet) {
							if err := peer.peerConnection.WriteRTCP(pkts); err != nil {
								log.Printf("transport-cc ERROR | %s", err)
							}
						})
						track.OnCloseAsync(func() {
							_ = peer.Signal.DispatchOffer()
						})
					}
				}

				if err == nil && !exist {
					go peer.Signal.DispatchOffer()
				}

				if track == nil {
					return
				}

				// WriteRTP takes about 50µs
				track.WriteRTP(pkt)
			})
		}
	})
}

func (p *PeerContext) CreateDataChannel(label string, options *webrtc.DataChannelInit) (*webrtc.DataChannel, error) {
	return p.peerConnection.CreateDataChannel(label, options)
}

func (p *PeerContext) SetAnswer(desc webrtc.SessionDescription) error {
	return p.peerConnection.SetRemoteDescription(desc)
}

func (p *PeerContext) Offer() (offer string, err error) {
	if p.peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
		return "", ErrPeerConnectionClosed
	}

	offerSdp, err := p.peerConnection.CreateOffer(nil)
	if err != nil {
		return "", err
	}

	if err = p.peerConnection.SetLocalDescription(offerSdp); err != nil {
		return "", err
	}

	offerJson, err := json.Marshal(offerSdp)
	if err != nil {
		return "", err
	}

	return string(offerJson), nil
}

func (p *PeerContext) SetCandidate(candidate webrtc.ICECandidateInit) error {
	return p.peerConnection.AddICECandidate(candidate)
}

func (p *PeerContext) Close(err error) error {
	// TODO: May be leak of not closed/removed resources
	p.cancel(err)
	return p.peerConnection.Close()
}

func (p *PeerContext) AddTransceiver(kind []webrtc.RTPCodecType) error {
	for _, t := range kind {
		if _, err := p.peerConnection.AddTransceiverFromKind(t,
			webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionRecvonly,
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func (p *PeerContext) OnConnectionStateChange(f func(p webrtc.PeerConnectionState)) {
	p.peerConnection.OnConnectionStateChange(f)
}

func (p *PeerContext) Done() <-chan struct{} {
	return p.ctx.Done()
}

func (p *PeerContext) Err() error {
	return p.ctx.Err()
}

func (p *PeerContext) SetStats(stats *rtpstats.RtpStats) {
	p.stats = stats
}

func (p *PeerContext) newSubscriber() {
	c, cancel := context.WithCancelCause(p.ctx)
	subscriber := &Subscriber{
		peerId:         p.PeerID,
		peerConnection: p.peerConnection,
		loopback:       make(map[string]*LoopbackTrackContext),
		tracks:         make(map[string]*TrackContext),
		ctx:            c,
		cancel:         cancel,
	}
	p.Subscriber = subscriber
}

var _ ICEAgent = (*PeerContext)(nil)

func (p *PeerContext) newSignal(conn WebsocketWriter) {
	signal := newSignal(conn, p)
	p.Signal = signal
}

func (p *PeerContext) newPeerConnection() error {
	peerConnection, err := p.webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return err
	}
	p.peerConnection = peerConnection
	return nil
}

type NewPeerContextParams struct {
	Context context.Context
	WS      WebsocketWriter
	API     *webrtc.API
}

func NewPeerContext(params NewPeerContextParams) (*PeerContext, error) {
	ctx, cancel := context.WithCancelCause(params.Context)
	p := &PeerContext{
		PeerID: uuid.NewString(),
		ctx:    ctx,
		cancel: cancel,
		webrtc: params.API,
	}
	if err := p.newPeerConnection(); err != nil {
		return nil, err
	}
	p.newSubscriber()
	p.newSignal(params.WS)
	return p, nil
}