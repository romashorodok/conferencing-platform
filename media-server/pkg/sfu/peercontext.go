package sfu

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v4"
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
	spreader          trackSpreader

	publishTracks   map[string]*ActiveTrackContext
	publishTracksMu sync.Mutex
}

type trackWritable interface {
	OnCloseAsync(func())
	GetTrackWriterRTP() (TrackWriterRTP, error)
}

type trackSpreader interface {
	TrackDownToPeers(*TrackContext) error
	TrackDownStopToPeers(context.Context, *TrackContext) error

	SanitizePeerSenders(*PeerContext) error
}

func (p *PeerContext) WatchTrack(track *ActiveTrackContext) {
	t := track.trackContext
	bus := t.TrackObserver()
	defer t.TrackObserverUnref(bus)
	// defer log.Println("stop watch track. Peer", p.PeerID, "for track", track.trackContext.ID())
	// log.Println("start watch track. Peer", p.PeerID, "for track", track.trackContext.ID())

	for {
		select {
		case <-t.Done():
			// log.Println("Watch track context doen for", t.ID())
			p.Subscriber.DeleteTrack(track)
			p.Signal.DispatchOffer()
			// ack := p.Subscriber.DetachTrack(t)
			// err := <-ack.Result
			// log.Println("track context done for err", err)
			return
		case msg := <-bus:
			switch evt := msg.Unbox().(type) {
			case TrackContextMediaChange:
				log.Println("on track media change", evt.track.id)
				track, exist := p.Subscriber.HasTrack(evt.track.ID())
				if !exist {
					log.Println("TrackContextMediaChange |  Media changed for not attached track. Not found active track")
					continue
				}
				err := track.SwitchActiveTrackMedia(p.peerConnection)
				log.Println("On switch active track media", err)
				p.Signal.DispatchOffer()
			}
		}
	}
}

func (p *PeerContext) WatchSubscriber(sub *Subscriber) {
	bus := sub.Observer()
	defer sub.ObserverUnref(bus)

	for {
		select {
		case <-sub.Done():
			// log.Println("watch subscriber done")
			return
		case msg := <-bus:
			switch evt := msg.Unbox().(type) {
			case SubscriberTrackAttached:
				go p.WatchTrack(evt.ActiveTrack())
				p.Signal.DispatchOffer()
			case SubscriberTrackDetached:
				log.Println("Get detach event")
			}
		}
	}
}

func (p *PeerContext) OnTrack() {
	streamID := uuid.NewString()

	go p.Subscriber.WatchTrackAttach()
	go p.Subscriber.WatchTrackDetach()

	go p.WatchSubscriber(p.Subscriber)

	p.peerConnection.OnTrack(func(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
		filter := FILTER_NONE

		ack := p.Subscriber.Track(streamID, t, recv, filter)
		select {
		case <-p.Done():
			return
		case err := <-ack.Result:
			if err != nil {
				log.Println("Unable attach track to subscriber. Err:", err)
				err = errors.Join(err, ErrUnsupportedTrack)
				_ = p.Signal.conn.WriteJSON(err)
				_ = p.Close(err)
				return
			}
		}

		activeTrack, exist := p.Subscriber.HasTrack(ack.TrackContext.ID())
		if !exist {
			err := ErrTrackNotFound
			log.Println("Unable found subscriber track. Err:", err)
			_ = p.Signal.conn.WriteJSON(err)
			_ = p.Close(err)
			return
		}

		p.publishTrack(activeTrack)

		tctx := ack.TrackContext
		var track trackWritable = tctx

		p.Signal.DispatchOffer()

		_ = TrackContextRegistry.Add(tctx)

		go p.spreader.TrackDownToPeers(tctx)
		go p.spreader.TrackDownStopToPeers(p.ctx, tctx)

		go func() {
			_ = p.spreader.SanitizePeerSenders(p)
			// log.Println("track up for peer??", err)
		}()

		log.Println("On track", t.ID())

		// log.Printf("%+v", p.Subscriber.tracks)

		for {
			select {
			case <-p.ctx.Done():
				return
			default:
			}

			pkt, _, err := t.ReadRTP()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// p.Subscriber.DeleteTrack(tctx.id)
					p.publishTrackDelete(activeTrack)
					TrackContextRegistry.Remove(tctx)
					err = tctx.Close()
					p.Signal.DispatchOffer()
					return
				}
				continue
			}

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

func (p *PeerContext) GetVideoTrackContext() (*ActiveTrackContext, error) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()

	for _, pub := range p.publishTracks {
		if pub.trackContext.codecKind == webrtc.RTPCodecTypeVideo {
			return pub, nil
		}
	}

	return nil, ErrTrackNotFound
}

func (p *PeerContext) GetAudioTrackContext() (*ActiveTrackContext, error) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()

	for _, pub := range p.publishTracks {
		if pub.trackContext.codecKind == webrtc.RTPCodecTypeAudio {
			return pub, nil
		}
	}

	return nil, ErrTrackNotFound
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

	var track *ActiveTrackContext
	err = nil
	switch mimeType {
	case MIME_TYPE_VIDEO:
		track, err = p.GetVideoTrackContext()
	case MIME_TYPE_AUDIO:
		track, err = p.GetAudioTrackContext()
	default:
		return errors.New("unknown mime type")
	}
	if err != nil {
		log.Println("Unable switch filter. Not found publish track context")
		return err
	}

	if err = track.trackContext.SetFilter(filter); err != nil {
		log.Println("set filter error", err)
		return err
	}

	// _ = p.peerConnection.RemoveTrack(track.LoadSender())
	//
	// sender, err := p.peerConnection.AddTrack(track.trackContext.GetLocalTrack())
	// if err != nil {
	// 	log.Println("unable add track on switch filter", err)
	// 	return err
	// }
	// track.StoreSender(sender)
	// p.Signal.DispatchOffer()

	return nil
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

func (p *PeerContext) publishTrack(t *ActiveTrackContext) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()
	p.publishTracks[t.trackContext.ID()] = t
}

func (p *PeerContext) publishTrackDelete(t *ActiveTrackContext) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()
	delete(p.publishTracks, t.trackContext.ID())
}

func (p *PeerContext) newSubscriber() {
	c, cancel := context.WithCancelCause(p.ctx)
	subscriber := &Subscriber{
		webrtc:           p.webrtc,
		peerId:           p.PeerID,
		peerConnection:   p.peerConnection,
		pipeAllocContext: p.pipeAllocContext,
		// loopback:       make(map[string]*LoopbackTrackContext),
		immediateAttachTrack: make(chan watchTrackAck),
		immediateDetachTrack: make(chan watchTrackAck),

		ctx:    c,
		cancel: cancel,
	}
	obs := make([]chan SubscriberMessage[any], 0)
	subscriber.observers.Store(&obs)
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
	Spreader         trackSpreader
}

func NewPeerContext(params NewPeerContextParams) (*PeerContext, error) {
	ctx, cancel := context.WithCancelCause(params.Context)
	p := &PeerContext{
		PeerID:           uuid.NewString(),
		ctx:              ctx,
		cancel:           cancel,
		webrtc:           params.API,
		pipeAllocContext: params.PipeAllocContext,
		publishTracks:    make(map[string]*ActiveTrackContext),
		spreader:         params.Spreader,
	}
	if err := p.newPeerConnection(); err != nil {
		return nil, err
	}
	p.newSubscriber()
	p.newSignal(params.WS)
	return p, nil
}
