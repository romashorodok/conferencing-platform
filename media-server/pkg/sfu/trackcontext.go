package sfu

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/pion/rtp"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type TrackWriterRTP interface {
	WriteRTP(*rtp.Packet) error
}

type TrackRemoteWriterSample interface {
	WriteRemote(sample media.Sample) error
}

type TrackWriter interface {
	TrackWriterRTP
	TrackRemoteWriterSample
	GetLocalTrack() webrtc.TrackLocal
}

var (
	ErrUnsupportedCaps    = errors.New("")
	ErrBadTrackAllocation = errors.New("")
)

type TrackMediaEngineRtp struct {
	rtp *webrtc.TrackLocalStaticRTP
}

func (t *TrackMediaEngineRtp) WriteRTP(pkt *rtp.Packet) error {
	return t.rtp.WriteRTP(pkt)
}

func (t *TrackMediaEngineRtp) WriteRemote(sample media.Sample) error {
	return errors.Join(ErrUnsupportedCaps, errors.New("unable write into rtp. Use WriteRTP instead"))
}

func (t *TrackMediaEngineRtp) GetLocalTrack() webrtc.TrackLocal {
	return t.rtp
}

var _ TrackWriter = (*TrackMediaEngineRtp)(nil)

func NewTrackWriterRtp(codecCaps webrtc.RTPCodecCapability, id, streamID string) (*TrackMediaEngineRtp, error) {
	rtp, err := webrtc.NewTrackLocalStaticRTP(codecCaps, id, streamID)
	if err != nil {
		return nil, err
	}
	return &TrackMediaEngineRtp{
		rtp: rtp,
	}, nil
}

type TrackContextMediaChange struct {
	track *TrackContext
}

type TrackContextClose struct{}

type TrackContextEvent interface {
	TrackContextMediaChange | TrackContextClose
}

type TrackContextMessage[F any] struct {
	value F
}

func (m *TrackContextMessage[F]) Unbox() F {
	return m.value
}

func NewTrackContextMessage[F TrackContextEvent](evt F) TrackContextMessage[any] {
	return TrackContextMessage[any]{
		value: evt,
	}
}

type TrackContext struct {
	webrtc       *webrtc.API
	id           string
	streamID     string
	SourcePeerID string

	rid         string
	ssrc        webrtc.SSRC
	payloadType webrtc.PayloadType

	media   TrackWriter
	mediaMu sync.Mutex

	codecParams webrtc.RTPCodecParameters
	codecKind   webrtc.RTPCodecType

	peerConnection *webrtc.PeerConnection
	transceiver    *webrtc.RTPTransceiver
	sender         *webrtc.RTPSender

	observers   []chan TrackContextMessage[any]
	observersMu sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
}

func (t *TrackContext) GetTrackWriterRTP() (TrackWriterRTP, error) {
	if t.media == nil {
		return nil, errors.New("track media engine empty")
	}

	return t.media, nil
}

func (t *TrackContext) GetTrackRemoteWriterSample() (TrackRemoteWriterSample, error) {
	if t.media == nil {
		return nil, errors.New("track media engine empty")
	}

	switch t.media.(type) {
	case *TrackMediaEngineRtp:
		return nil, ErrUnsupportedCaps
	}

	return t.media, nil
}

func (t *TrackContext) Close() (err error) {
	log.Println("[Close] track context close ID:", t.ID())
	t.cancel()
	return
}

func (t *TrackContext) OnCloseAsync(f func()) {
	go func() {
		select {
		case <-t.ctx.Done():
			f()
		}
	}()
}

func (t *TrackContext) Done() <-chan struct{} {
	return t.ctx.Done()
}

func (t *TrackContext) DoneErr() error {
	return t.ctx.Err()
}

func (t *TrackContext) ID() string {
	return t.id
}

func (t *TrackContext) StreamID() string {
	return t.streamID
}

func (t *TrackContext) GetClockRate() uint32 {
	return t.codecParams.ClockRate
}

func (t *TrackContext) GetLocalTrack() webrtc.TrackLocal {
	t.mediaMu.Lock()
	defer t.mediaMu.Unlock()
	return t.media.GetLocalTrack()
}

func (t *TrackContext) dispatch(msg TrackContextMessage[any]) {
	t.observersMu.Lock()
	defer t.observersMu.Unlock()

	for _, ch := range t.observers {
		go func(c chan TrackContextMessage[any]) {
			select {
			case <-t.Done():
				return
			case c <- msg:
				return
			}
		}(ch)
	}
}

func (t *TrackContext) TrackObserver() <-chan TrackContextMessage[any] {
	t.observersMu.Lock()
	defer t.observersMu.Unlock()

	ch := make(chan TrackContextMessage[any])
	t.observers = append(t.observers, ch)
	return ch
}

func (t *TrackContext) TrackObserverUnref(obs <-chan TrackContextMessage[any]) {
	t.observersMu.Lock()
	defer t.observersMu.Unlock()

	for i, observer := range t.observers {
		if obs == observer {
			close(observer)
			t.observers = append(t.observers[:i], t.observers[i+1:]...)
			return
		}
	}
}

type NewTrackContextParams struct {
	SourcePeerID string
	ID           string
	StreamID     string
	RID          string
	SSRC         webrtc.SSRC
	PayloadType  webrtc.PayloadType

	CodecParams    webrtc.RTPCodecParameters
	Kind           webrtc.RTPCodecType
	API            *webrtc.API
	PeerConnection *webrtc.PeerConnection
}

func NewTrackContext(ctx context.Context, params NewTrackContextParams) *TrackContext {
	c, cancel := context.WithCancel(ctx)

	media, err := NewTrackWriterRtp(params.CodecParams.RTPCodecCapability, params.ID, params.StreamID)
	if err != nil {
		cancel()
		return nil
	}

	trackContext := &TrackContext{
		SourcePeerID: params.SourcePeerID,
		webrtc:       params.API,
		id:           params.ID,
		streamID:     params.StreamID,
		rid:          params.RID,
		ssrc:         params.SSRC,
		payloadType:  params.PayloadType,
		media:        media,

		codecParams:    params.CodecParams,
		codecKind:      params.Kind,
		peerConnection: params.PeerConnection,

		observers: make([]chan TrackContextMessage[any], 0),
		ctx:       c,
		cancel:    cancel,
	}

	return trackContext
}

type ActiveTrackContext struct {
	trackContext *TrackContext
	sender       atomic.Pointer[webrtc.RTPSender]
	transceiver  atomic.Pointer[webrtc.RTPTransceiver]
}

func (a *ActiveTrackContext) LoadSender() *webrtc.RTPSender {
	return a.sender.Load()
}

func (a *ActiveTrackContext) StoreSender(s *webrtc.RTPSender) {
	a.sender.Store(s)
}

func (a *ActiveTrackContext) LoadTransiver() *webrtc.RTPTransceiver {
	return a.transceiver.Load()
}

func (a *ActiveTrackContext) StoreTransiver(transiv *webrtc.RTPTransceiver) {
	a.transceiver.Store(transiv)
}

func (a *ActiveTrackContext) SwitchActiveTrackMedia(api *webrtc.API, pc *webrtc.PeerConnection) error {
	sender := a.LoadSender()
	if sender == nil {
		return ErrSwitchActiveTrackNotFoundSender
	}

	transiv := a.LoadTransiver()
	if transiv == nil {
		return ErrSwitchActiveTrackNotFoundTransiv
	}

	nextSender, err := api.NewRTPSender(a.trackContext.GetLocalTrack(), pc.SCTP().Transport())
	if err != nil {
		return err
	}

	_ = sender.Stop()

	err = transiv.SetSender(nextSender, a.trackContext.GetLocalTrack())
	log.Println("SetSender err:", err)

	a.StoreSender(sender)

	return nil
}

func NewActiveTrackContext(transiv *webrtc.RTPTransceiver, sender *webrtc.RTPSender, track *TrackContext) *ActiveTrackContext {
	a := &ActiveTrackContext{trackContext: track}
	a.StoreSender(sender)
	a.StoreTransiver(transiv)
	return a
}

type PublishTrackContext struct {
	trackContext *TrackContext
}

func NewPublishTrackContext(t *TrackContext) *PublishTrackContext {
	return &PublishTrackContext{
		trackContext: t,
	}
}
