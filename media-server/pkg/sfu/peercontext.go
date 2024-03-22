package sfu

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/rtpstats"
)

type PeerContext struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	PeerID            protocol.PeerID
	webrtc            *webrtc.API
	peerConnection    *webrtc.PeerConnection
	stats             *rtpstats.RtpStats
	Signal            *Signal
	Subscriber        *Subscriber
	videoTrackContext *TrackContext
	audioTrackContext *TrackContext
	pipeAllocContext  *AllocatorsContext
}

type trackWritable interface {
	OnCloseAsync(func())
	GetTrackWriterRTP() (TrackWriterRTP, error)
}

func (p *PeerContext) OnTrack(peerContextPool *PeerContextPool) {
	p.peerConnection.OnTrack(func(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
		// defer func() {
		// 	for _, peer := range pool.Get() {
		// 		_ = peer.Subscriber.DeleteTrack(t.ID())
		// 	}
		// }()

		filter := FILTER_NONE

		tctx, err := p.Subscriber.Track(t, recv, filter)
		if err != nil {
			err = errors.Join(err, ErrUnsupportedTrackCodec)
			_ = p.Signal.conn.WriteJSON(err)
			_ = p.Close(err)
			return
		}

		defer func() {
			_ = tctx.Close()
		}()
		p.Signal.DispatchOffer()

		var track trackWritable
		switch t.Codec().MimeType {
		case webrtc.MimeTypeOpus:
			// time.Sleep(time.Hour)
			p.audioTrackContext = tctx
			track = NewTrackContextOpus(tctx)
		case webrtc.MimeTypeVP8:
			p.videoTrackContext = tctx
			track = NewTrackContextVp8(tctx)
			// pipe := pipelines.NewPipelineDummy(t.Codec().RTPCodecCapability)
			// _ = tctx.SetPipelines([]*pipelines.PipelineDummy{
			// 	pipe,
			// })
		default:
			_ = p.Signal.conn.WriteJSON(ErrUnsupportedTrackCodec)
			_ = p.Close(ErrUnsupportedTrackCodec)
			return
		}

		// _ = p.trackContextPool.Add(tctx)

		// caps := t.Codec()
		// NOTE: pipe must have use rtp pkt input but in bytes format
		// pipe, _ := p.pipeAllocContext.Allocate(&AllocateParams{
		// 	TrackID:   tctx.ID(),
		// 	Filter:    *FILTER_RTP_VP8_DUMMY,
		// 	MimeType:  caps.MimeType,
		// 	ClockRate: caps.ClockRate,
		// })
		// pipe.Start()
		// pipe.

		_ = TrackContextRegistry.Add(tctx)
		defer TrackContextRegistry.Remove(tctx)

		// track.OnSample(func(sample *media.Sample) {
		// 	// pipe.Sink(sample.Data, sample.Timestamp, sample.Duration)
		// })

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

			// pktBytes, err := pkt.Marshal()
			// if err != nil {
			// 	continue
			// }

			// pipe.Sink(pktBytes, time.Time{}, -1)
			// pkt.Timestamp

			// _ = track.WriteRTP(pkt)

			writer, err := track.GetTrackWriterRTP()
			if err != nil {
				log.Println("unable get rtp writer. Err:", err)
				continue
			}

			err = writer.WriteRTP(pkt)
            if err != nil {
                log.Println("unable write rtp pkt. Err:", err)
                continue
            }
		}
	})
}

func (p *PeerContext) GetVideoTrackContext() *TrackContext {
	return p.videoTrackContext
}

func (p *PeerContext) GetAudioTrackContext() *TrackContext {
	return p.audioTrackContext
}

func (p *PeerContext) SwitchFilter(filterName string, mimeTypeName string) error {
	filter, err := p.pipeAllocContext.Filter(filterName)
	if err != nil {
		return err
	}

	var mimeType MimeType
	for _, mime := range filter.MimeTypes {
		if mimeTypeName == mime.String() {
			mimeType = mime
		}
	}
	if mimeType.String() == "" {
		return errors.New("unknown mime type")
	}

	var track *TrackContext
	switch mimeType {
	case MIME_TYPE_VIDEO:
		track = p.GetVideoTrackContext()
	case MIME_TYPE_AUDIO:
		track = p.GetAudioTrackContext()
	default:
		return errors.New("unknown mime type")
	}

	return track.SetFilter(filter)
}

type filterPayload struct {
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Enabled  bool   `json:"enabled"`
}
type filtersResult struct {
	Audio []filterPayload `json:"audio"`
	Video []filterPayload `json:"video"`
}

func (p *PeerContext) Filters() *filtersResult {
	availableFilters := p.pipeAllocContext.Filters()

	var audioFilters []filterPayload
	var videoFilters []filterPayload

	for _, filter := range availableFilters {
		for _, mimeType := range filter.MimeTypes {
			switch t := mimeType; t {
			case MIME_TYPE_VIDEO:
				videoFilters = append(videoFilters, filterPayload{
					Name:     filter.GetName(),
					MimeType: t.String(),
					Enabled:  false,
				})
			case MIME_TYPE_AUDIO:
				audioFilters = append(audioFilters, filterPayload{
					Name:     filter.GetName(),
					MimeType: t.String(),
					Enabled:  false,
				})
			default:
				continue
			}
		}
	}

	return &filtersResult{
		Audio: audioFilters,
		Video: videoFilters,
	}
}

// func (p *PeerContext) OnTrack(pool *PeerContextPool) {
// 	p.peerConnection.OnTrack(func(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
// 		// pipe, err := p.pipeAllocContext.Allocate(pipelines.RTP_VP8_BASE)
// 		// log.Println(pipe, err)
// 		// _ = err
// 		// pipe.Start()
// 		// pipe.Close()
//
// 		defer func() {
// 			for _, peer := range pool.Get() {
// 				_ = peer.Subscriber.DeleteTrack(t.ID())
// 			}
// 		}()
// 		log.Println("On track", t.ID())
//
// 		var threshold uint64 = 1000000
// 		var step uint64 = 2
//
// 		for {
// 			select {
// 			case <-p.ctx.Done():
// 				return
// 			default:
// 			}
//
// 			pkt, _, err := t.ReadRTP()
// 			if err != nil {
// 				if errors.Is(err, io.EOF) {
// 					return
// 				}
// 				continue
// 			}
//
// 			executils.ParallelExec(pool.Get(), threshold, step, func(peer *PeerContext) {
// 				select {
// 				case <-peer.ctx.Done():
// 					return
// 				default:
// 				}
//
// 				var track trackWritable
// 				var exist bool
// 				var err error
//
// 				switch {
// 				case p.PeerID == peer.PeerID:
// 					track, exist = peer.Subscriber.HasLoopbackTrack(t.ID())
// 					if !exist {
// 						log.Println("Create loopback track")
// 						track, err = peer.Subscriber.LoopbackTrack(t, recv)
// 						track.OnFeedback(func(pkts []rtcp.Packet) {
// 							if err = peer.peerConnection.WriteRTCP(pkts); err != nil {
// 								log.Printf("transport-cc ERROR | %s", err)
// 							}
// 						})
// 						track.OnCloseAsync(func() {
// 							_ = peer.Signal.DispatchOffer()
// 						})
// 					}
//
// 				default:
// 					track, exist = peer.Subscriber.HasTrack(t.ID())
// 					if !exist {
// 						log.Println("Create local track track")
// 						track, err = peer.Subscriber.Track(t, recv)
// 						track.OnFeedback(func(pkts []rtcp.Packet) {
// 							if err = peer.peerConnection.WriteRTCP(pkts); err != nil {
// 								log.Printf("transport-cc ERROR | %s", err)
// 							}
// 						})
// 						track.OnCloseAsync(func() {
// 							_ = peer.Signal.DispatchOffer()
// 						})
// 					}
// 				}
//
// 				if err == nil && !exist {
// 					go peer.Signal.DispatchOffer()
// 				}
//
// 				if track == nil {
// 					return
// 				}
//
// 				// WriteRTP takes about 50µs
// 				track.WriteRTP(pkt)
// 			})
// 		}
// 	})
// }

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
		peerId:           p.PeerID,
		peerConnection:   p.peerConnection,
		pipeAllocContext: p.pipeAllocContext,
		// loopback:       make(map[string]*LoopbackTrackContext),
		tracks: make(map[string]*TrackContext),
		ctx:    c,
		cancel: cancel,
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
	Context          context.Context
	WS               WebsocketWriter
	API              *webrtc.API
	PipeAllocContext *AllocatorsContext
}

func NewPeerContext(params NewPeerContextParams) (*PeerContext, error) {
	ctx, cancel := context.WithCancelCause(params.Context)
	p := &PeerContext{
		PeerID:           uuid.NewString(),
		ctx:              ctx,
		cancel:           cancel,
		webrtc:           params.API,
		pipeAllocContext: params.PipeAllocContext,
	}
	if err := p.newPeerConnection(); err != nil {
		return nil, err
	}
	p.newSubscriber()
	p.newSignal(params.WS)
	return p, nil
}
