package sfu

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/rtcerr"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/rtpstats"
)

func generateHash(data string) string {
	hash := sha256.New()
	hash.Write([]byte(data))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

type SessionDesc struct {
	negotiated atomic.Pointer[webrtc.SessionDescription]
	pending    atomic.Pointer[webrtc.SessionDescription]

	negotiatedHash atomic.Value /* string|nil */
	pendingHash    atomic.Value /* string|nil */

	retry           atomic.Bool
	sessionMu       sync.Mutex
	sessionMuLocked atomic.Bool

	ctxDeadline       atomic.Pointer[context.Context]
	ctxDeadlineCancel atomic.Pointer[context.CancelFunc]

	pctx context.Context
}

func (s *SessionDesc) LoadDeadlineContext() context.Context {
	val := s.ctxDeadline.Load()
	if val == nil {
		return nil
	}
	return *val
}

func (s *SessionDesc) StoreDeadlineContext(ctx context.Context, cancel context.CancelFunc) {
	s.ctxDeadline.Store(&ctx)
	s.ctxDeadlineCancel.Store(&cancel)
}

func (s *SessionDesc) SetPendingDesc(desc *webrtc.SessionDescription) error {
	log.Println("[SetPendingDesc] Try lock. Desc:", desc.Type)
	locked := s.sessionMu.TryLock()
loop:
	for !locked {
		deadline := s.LoadDeadlineContext()
		if deadline != nil {
			select {
			case <-deadline.Done():
				deadlineTime, _ := deadline.Deadline()
				log.Printf("[SetPendingDesc] Deadline exhausted at %s. Desc:%s", deadlineTime, desc.Type)
				s.sessionMu = sync.Mutex{}
			default:
			}
		}

		select {
		case <-s.pctx.Done():
			return ErrPeerConnectionClosed
		default:
			locked := s.sessionMu.TryLock()
			if locked {
				log.Println("[SetPendingDesc] Retry locked. Desc:", desc.Type)
				break loop
			}
			time.Sleep(time.Millisecond * 200)
			continue
		}
	}
	log.Println("[SetPendingDesc] Already locked. Desc:", desc.Type)
	s.sessionMuLocked.Store(true)

	ctxDeadline, cancel := context.WithDeadline(s.pctx, time.Now().Add(time.Second))
	s.StoreDeadlineContext(ctxDeadline, cancel)

	s.pendingHash.Store(generateHash(desc.SDP))
	s.pending.Store(desc)
	s.retry.Store(true)

	return nil
}

var EmptyOfferHash string = ""

func (s *SessionDesc) Submit(offerHash string) error {
	pending := s.pending.Load()

	if pending == nil {
		log.Println("[SessionDescSubmit] Pending nil")
		return ErrSubmitEmptyPendingSessionDesc
	}

	if offerHash == "" {
		log.Println("[SessionDescSubmit] offerHash empty")
		return ErrSubmitOfferStateEmpty
	}

	swapped := s.pendingHash.CompareAndSwap(offerHash, EmptyOfferHash)
	if !swapped {
		log.Println("[SessionDescSubmit] Unable unlock state")
		return ErrSubmitOfferRaceCondition
	}

	//

	s.negotiatedHash.Store(offerHash)

	negotiated := s.pending.Swap(nil)
	s.negotiated.Store(negotiated)
	s.retry.Store(false)

	if s.sessionMuLocked.Load() {
		s.sessionMu.Unlock()
	}
	return nil
}

func (s *SessionDesc) GetPending() *webrtc.SessionDescription {
	return s.pending.Load()
}

func (s *SessionDesc) GetRetry() bool {
	return s.retry.Load()
}

func NewSessionDesc(ctx context.Context) *SessionDesc {
	return &SessionDesc{
		pctx: ctx,
	}
}

type PeerContext struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	peerID          string
	webrtc          *webrtc.API
	peerConnection  *webrtc.PeerConnection
	stats           *rtpstats.RtpStats
	offer           *SessionDesc
	Signal          *Signal
	Subscriber      *Subscriber
	transceiverPool *TransceiverPool
	spreader        trackSpreader

	publishTracks   map[string]*PublishTrackContext
	publishTracksMu sync.Mutex
}

type trackWritable interface {
	OnCloseAsync(func())
	GetTrackWriterRTP() (TrackWriterRTP, error)
}

type trackSpreader interface {
	TrackDownToPeers(*PeerContext, *TrackContext) error
	TrackDownStopToPeers(*PeerContext, *TrackContext) error

	SanitizePeerSenders(*PeerContext) error
	PeerPublishingSenders(peerTarget *PeerContext) map[string]OptionalSenderBox[any]
}

func (p *PeerContext) ObserveTrack(track *ActiveTrackContext) {
	t := track.trackContext
	bus := t.TrackObserver()
	defer t.TrackObserverUnref(bus)

	for {
		select {
		case <-t.Done():
			p.Subscriber.DeleteTrack(track)
			p.spreader.SanitizePeerSenders(p)
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
				err := track.SwitchActiveTrackMedia(p.webrtc, p.peerConnection)
				log.Println("On switch active track media", err)
				p.spreader.SanitizePeerSenders(p)
			}
		}
	}
}

func (p *PeerContext) ObserveSubscriber(sub *Subscriber) {
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
				log.Println("Track attached", evt.track.trackContext.ID())
				go p.ObserveTrack(evt.ActiveTrack())
				p.spreader.SanitizePeerSenders(p)
			case SubscriberTrackDetached:
				p.spreader.SanitizePeerSenders(p)
				log.Println("Get detach event")
			}
		}
	}
}

func (p *PeerContext) OnTrack() {
	var onTrackMu sync.Mutex
	pubStreamID := uuid.NewString()

	go p.Subscriber.HandleTrackAttach()
	go p.Subscriber.HandleTrackDetach()

	go p.ObserveSubscriber(p.Subscriber)

	onCloseTrack := func(err error, message ...any) {
		onTrackMu.Unlock()
		log.Println(message...)
		err = errors.Join(err, ErrUnsupportedTrack)
		_ = p.Signal.conn.WriteJSON(err)
		_ = p.Close(err)
	}

	p.peerConnection.OnTrack(func(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
		onTrackMu.Lock()

		log.Println("On track - ID:", t.ID(), "SSRC:", t.SSRC(), "StreamID:", t.StreamID())
		tctx := p.Subscriber.Track(pubStreamID, t, recv)

		ptctx := NewPublishTrackContext(tctx)
		p.publishTrack(ptctx)
		defer p.publishTrackDelete(ptctx)

		ack := p.Subscriber.AttachTrack(tctx)
		select {
		case <-p.Done():
			onTrackMu.Unlock()
			return
		case err := <-ack.Result:
			if err != nil {
				onCloseTrack(err, "[OnTrack] Unable attach track to subscriber. Err:", err)
				return
			}
		}

		TrackContextRegistry.Add(tctx)
		defer TrackContextRegistry.Remove(tctx)

		err := p.spreader.TrackDownToPeers(p, tctx)
		if err != nil {
			onCloseTrack(err, "[OnTrack] Unable down", tctx.ID(), "track. Err:", err)
			return
		}
		defer p.spreader.TrackDownStopToPeers(p, tctx)

		onTrackMu.Unlock()

		var track trackWritable = ack.TrackContext

		for {
			select {
			case <-p.ctx.Done():
				return
			default:
			}

			pkt, _, err := t.ReadRTP()
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Printf("[EOF] Publish sender track ID: %s", t.ID())
					_ = tctx.Close()
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

func (p *PeerContext) GetVideoPublishTrack() (*PublishTrackContext, error) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()

	for _, pub := range p.publishTracks {
		if pub.trackContext.codecKind == webrtc.RTPCodecTypeVideo {
			return pub, nil
		}
	}

	return nil, ErrTrackNotFound
}

func (p *PeerContext) GetAudioPublishTrack() (*PublishTrackContext, error) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()

	for _, pub := range p.publishTracks {
		if pub.trackContext.codecKind == webrtc.RTPCodecTypeAudio {
			return pub, nil
		}
	}

	return nil, ErrTrackNotFound
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

func (p *PeerContext) SynchronizeOfferState() {
	ticker := time.NewTicker(time.Second * 4)
	for {
		select {
		case <-p.Done():
			return
		case <-ticker.C:
			senders := p.spreader.PeerPublishingSenders(p)
			// log.Printf("[SynchronizeOfferState] try sanitize %s", p.PeerID())
			// log.Printf("PeerPublishingSenders: %+v", senders)
			// log.Printf("GetSenders: %+v", p.peerConnection.GetSenders())
			for _, sender := range senders {
				switch s := sender.value.(type) {
				case UnattachedSender:
					log.Printf("[SynchronizeOfferState] found unattached sender. track ID: %s", s.track.trackContext.ID())
					ack := p.Subscriber.AttachTrack(s.track.trackContext)
					if err := <-ack.Result; err != nil {
						log.Println("attach track err", err)
					}
				default:
				}
			}
		}
	}
}

func (p *PeerContext) CreateDataChannel(label string, options *webrtc.DataChannelInit) (*webrtc.DataChannel, error) {
	return p.peerConnection.CreateDataChannel(label, options)
}

func (p *PeerContext) SetAnswer(desc webrtc.SessionDescription) error {
	err := p.peerConnection.SetRemoteDescription(desc)
	if err != nil {
		return err
	}
	return nil
}

type CommitOfferStateMessage struct {
	StateHash string `json:"state_hash"`
}

func (p *PeerContext) CommitOfferState(msg CommitOfferStateMessage) error {
	return p.offer.Submit(msg.StateHash)
}

type offerResult struct {
	webrtc.SessionDescription

	HashState string `json:"hash_state"`
}

func (p *PeerContext) Offer() (offer string, err error) {
	if p.peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
		return "", ErrPeerConnectionClosed
	}

	offerSdp, err := p.peerConnection.CreateOffer(nil)
	if err != nil {
		return "", err
	}

	if err = p.offer.SetPendingDesc(&offerSdp); err != nil {
		return "", err
	}

	err = p.peerConnection.SetLocalDescription(offerSdp)
	if err != nil {
		switch e := err.(type) {
		case *rtcerr.InvalidModificationError:
			log.Println("GetPending old pending offer")
			pendingSDP := p.offer.GetPending()
			if pendingSDP == nil {
				return "", err
			}

			offerSdp = *pendingSDP
		default:
			log.Println("default error for local desc", reflect.TypeOf(e))
			return "", err
		}
	}

	offerJson, err := json.Marshal(offerResult{
		SessionDescription: offerSdp,
		HashState:          p.offer.pendingHash.Load().(string),
	})
	if err != nil {
		return "", err
	}

	return string(offerJson), nil
}

func (p *PeerContext) SetCandidate(candidate webrtc.ICECandidateInit) error {
	return p.peerConnection.AddICECandidate(candidate)
}

func (p *PeerContext) OnICECandidate(f func(*webrtc.ICECandidate)) {
	p.peerConnection.OnICECandidate(f)
}

func (p *PeerContext) Close(err error) error {
	// TODO: May be leak of not closed/removed resources
	p.cancel(err)
	return p.peerConnection.Close()
}

// NOTE: Chrome 122 has introduced bug with sendonly transceiver,
// to expect normal behavior I think better use shadow stream/tracks,
// also I have sanitize logic to check publish tracks
func (p *PeerContext) AddTransceiver(kinds []webrtc.RTPCodecType) error {
	streamID := "inactive"

	for _, t := range kinds {
		trackID := uuid.NewString()

		transiv, err := p.peerConnection.AddTransceiverFromKind(t,
			webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendrecv,
			},
		)
		if err != nil {
			return err
		}

		senderTrackCodec := transiv.Sender().GetParameters().RTPParameters.Codecs[0]
		track, err := webrtc.NewTrackLocalStaticRTP(senderTrackCodec.RTPCodecCapability, trackID, streamID)
		transiv.Sender().ReplaceTrack(track)
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

func (p *PeerContext) PeerID() string {
	return p.peerID
}

func (p *PeerContext) publishTrack(t *PublishTrackContext) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()
	p.publishTracks[t.trackContext.ID()] = t
}

func (p *PeerContext) publishTrackDelete(t *PublishTrackContext) {
	p.publishTracksMu.Lock()
	defer p.publishTracksMu.Unlock()
	delete(p.publishTracks, t.trackContext.ID())
}

func (p *PeerContext) newSubscriber() {
	c, cancel := context.WithCancelCause(p.ctx)
	subscriber := &Subscriber{
		webrtc:          p.webrtc,
		peerId:          p.peerID,
		peerConnection:  p.peerConnection,
		transceiverPool: p.transceiverPool,
		busAttachTrack:  make(chan watchTrackAck),
		busDetachTrack:  make(chan watchTrackAck),
		ctx:             c,
		cancel:          cancel,
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
	Context  context.Context
	WS       WebsocketWriter
	API      *webrtc.API
	Spreader trackSpreader
}

func NewPeerContext(params NewPeerContextParams) (*PeerContext, error) {
	ctx, cancel := context.WithCancelCause(params.Context)
	p := &PeerContext{
		peerID:          uuid.NewString(),
		ctx:             ctx,
		cancel:          cancel,
		webrtc:          params.API,
		publishTracks:   make(map[string]*PublishTrackContext),
		offer:           NewSessionDesc(ctx),
		transceiverPool: NewTransceiverPool(),
		spreader:        params.Spreader,
	}
	if err := p.newPeerConnection(); err != nil {
		return nil, err
	}
	p.newSubscriber()
	p.newSignal(params.WS)
	return p, nil
}
