package sfu

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

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
	SetPipeline(pipe Pipeline) error
	GetLocalTrack() webrtc.TrackLocal
}

var (
	ErrUnsupportedCaps    = errors.New("")
	ErrBadTrackAllocation = errors.New("")
)

type TrackMediaEngineRtp struct {
	rtp *webrtc.TrackLocalStaticRTP
}

func (t *TrackMediaEngineRtp) SetPipeline(pipe Pipeline) error {
	return errors.Join(ErrUnsupportedCaps, errors.New("unable set pipe"))
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

type TrackMediaEngineSample struct {
	sample *webrtc.TrackLocalStaticSample
	pipe   Pipeline
}

func (t *TrackMediaEngineSample) SetPipeline(pipe Pipeline) error {
	t.pipe = pipe
	return nil
}

func (t *TrackMediaEngineSample) WriteRTP(pkt *rtp.Packet) error {
	pktBytes, err := pkt.Marshal()
	if err != nil {
		return err
	}
	return t.pipe.Sink(pktBytes, time.Time{}, -1)
}

func (t *TrackMediaEngineSample) WriteRemote(sample media.Sample) error {
	return t.sample.WriteSample(sample)
}

func (t *TrackMediaEngineSample) GetLocalTrack() webrtc.TrackLocal {
	return t.sample
}

var _ TrackWriter = (*TrackMediaEngineSample)(nil)

func NewTrackWriterSample(codecCaps webrtc.RTPCodecCapability, id, streamID string) (*TrackMediaEngineSample, error) {
	sample, err := webrtc.NewTrackLocalStaticSample(codecCaps, id, streamID)
	if err != nil {
		return nil, err
	}

	return &TrackMediaEngineSample{
		sample: sample,
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
	webrtc   *webrtc.API
	id       string
	streamID string

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

	// rtp    *webrtc.TrackLocalStaticRTP
	// sample *webrtc.TrackLocalStaticSample

	filter           *Filter
	pipeAllocContext *AllocatorsContext

	observers   []chan TrackContextMessage[any]
	observersMu sync.Mutex

	// pipes     []Pipeline
	// sampleBus chan *media.Sample

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

// func (t *TrackContext) WriteSample(sample media.Sample) error {
// 	// return t.sample.WriteSample(sample)
// }

// func (t *TrackContext) WriteRTP(rtp *rtp.Packet) error {
// 	// return t.rtp.WriteRTP(rtp)
// }

// func (t *TrackContext) passThroughSink(sample *media.Sample) error {
// 	frame := sample.Data
// 	timestamp := sample.Timestamp
// 	duration := sample.Duration
// 	var err error
//
// 	for _, pipe := range t.pipes {
//         pipe.Sink(frame []byte, timestamp time.Time, duration time.Duration)
// 		// frame, timestamp, duration, err = pipe.Sink(frame, timestamp, duration)
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	sample.Data = frame
// 	t.sampleBus <- sample
// 	return nil
// }

// func (t *TrackContext) OnSample(f func(*media.Sample)) {
// 	go func() {
// 		for {
// 			select {
// 			case <-t.ctx.Done():
// 				return
// 			case sample := <-t.sampleBus:
// 				f(sample)
// 			}
// 		}
// 	}()
// }

// func (t *TrackContext) SetPipelines(pipes []*pipelines.PipelineDummy) {
// 	if pipes == nil || len(pipes) <= 0 {
// 		t.pipes = []*pipelines.PipelineDummy{
// 			pipelines.NewPipelineDummy(),
// 		}
// 		return
// 	}
// 	t.pipes = pipes
// }

func (t *TrackContext) Close() (err error) {
	log.Println("close track", t.ID())
	t.cancel()

	select {
	case <-t.Done():
		log.Println("Valid close 1")
	}

	select {
	case <-t.Done():
		log.Println("Valid close 2")
	}

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

func (t *TrackContext) SetFilter(filter *Filter) error {
	// if t.rtp.Kind() != t.sample.Kind() {
	// 	log.Panicf("different track mime type on context. RTP: %s SAMPLE: %s", t.rtp.Kind(), t.sample.Kind())
	// 	os.Exit(-1)
	// }

	// if t.rtp.Kind() != t.sender.Track().Kind() {
	// 	return errors.New("not allowed replace mime type of the track. Create a new one instead")
	// }

	var found bool
	switch kind := t.codecKind; kind {
	case webrtc.RTPCodecTypeAudio:
		for _, mimeType := range filter.MimeTypes {
			if kind.String() == mimeType.String() {
				found = true
				break
			}
		}
		if !found {
			return errors.New("unable switch track. Not found filter mime type")
		}

	case webrtc.RTPCodecTypeVideo:
		for _, mimeType := range filter.MimeTypes {
			if kind.String() == mimeType.String() {
				found = true
				break
			}
		}
		if !found {
			return errors.New("unable switch track. Not found filter mime type")
		}

	default:
		return errors.New("filter mime type mismatch")
	}

	var media TrackWriter
	var err error

	switch filter {
	case FILTER_NONE:
		media, err = NewTrackWriterRtp(t.codecParams.RTPCodecCapability, t.ID(), t.StreamID())
	case FILTER_RTP_VP8_DUMMY:
		media, err = NewTrackWriterSample(t.codecParams.RTPCodecCapability, t.ID(), t.StreamID())
		if err != nil {
			return err
		}

		log.Printf("pipe alloc context %+v", t.pipeAllocContext)
		// TODO: STOP the pipe if err exist
		pipe, _ := t.pipeAllocContext.Allocate(&AllocateParams{
			TrackID:   t.ID(),
			Filter:    FILTER_RTP_VP8_DUMMY,
			MimeType:  t.codecParams.MimeType,
			ClockRate: t.codecParams.ClockRate,
		})
		pipe.Start()
		// if err != nil {
		// 	return err
		// }
		log.Printf("pipe: %+v", pipe)
		_ = media.SetPipeline(pipe)
	default:
		media, err = NewTrackWriterRtp(t.codecParams.RTPCodecCapability, t.ID(), t.StreamID())
	}

	if err != nil {
		return errors.Join(ErrBadTrackAllocation, errors.New("unable create track for filter"))
	}

	// trans, err := t.createTransiverIfNotExist(media.GetLocalTrack())
	// if err != nil {
	// 	log.Println("trans err", err)
	// 	return err
	// }

	// if t.sender != nil {
	// 	if err = t.peerConnection.RemoveTrack(t.sender); err != nil {
	// 		log.Println("err track")
	// 		return err
	// 	}
	// }
	//
	// sender, err := t.peerConnection.AddTrack(media.GetLocalTrack())
	// if err != nil {
	// 	log.Println("err add track")
	// }
	//
	// t.sender = sender

	//
	// t.media.GetLocalTrack()

	//    t.media.GetLocalTrack()
	//    t.
	//
	//    t.peerConnection.AddTrack()
	//
	// if err := trans.Sender().ReplaceTrack(media.GetLocalTrack()); err != nil {
	// 	log.Println("sender err", err)
	// 	return err
	// }

	// if err := trans.SetSender(trans.Sender(), media.GetLocalTrack()); err != nil {
	// 	log.Println("sender err", err)
	// 	return err
	// }

	// t.webrtc.NewRTPSender(track webrtc.TrackLocal, transport *webrtc.DTLSTransport)

	// if t.peerConnection.SCTP().Transport()

	// if err := t.sender.ReplaceTrack(media.GetLocalTrack()); err != nil {
	// 	return err
	// }
	// log.Printf("replace track %+v", media)

	t.mediaMu.Lock()
	t.filter = filter
	t.media = media
	t.dispatch(NewTrackContextMessage(TrackContextMediaChange{
        track: t,
    }))
	t.mediaMu.Unlock()
	return nil
}

func (t *TrackContext) Filter() *Filter {
	return t.filter
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
	ID          string
	StreamID    string
	RID         string
	SSRC        webrtc.SSRC
	PayloadType webrtc.PayloadType

	CodecParams webrtc.RTPCodecParameters
	Kind        webrtc.RTPCodecType
	// TrackSample *webrtc.TrackLocalStaticSample
	// TrackRtp    *webrtc.TrackLocalStaticRTP

	// TODO: add alloc to filter
	PipeAllocContext *AllocatorsContext
	// Sender           *webrtc.RTPSender
	Filter         *Filter
	API            *webrtc.API
	PeerConnection *webrtc.PeerConnection
}

// TODO: creating of RTPSender must be here not on sub side
func NewTrackContext(ctx context.Context, params NewTrackContextParams) *TrackContext {
	if params.Filter == nil {
		params.Filter = FILTER_NONE
	}

	c, cancel := context.WithCancel(ctx)

	trackContext := &TrackContext{
		webrtc:      params.API,
		id:          params.ID,
		streamID:    params.StreamID,
		rid:         params.RID,
		ssrc:        params.SSRC,
		payloadType: params.PayloadType,

		codecParams:    params.CodecParams,
		codecKind:      params.Kind,
		peerConnection: params.PeerConnection,

		// rtp:      params.TrackRtp,
		// sample:   params.TrackSample,

		// track:  params.Track,
		pipeAllocContext: params.PipeAllocContext,

		observers: make([]chan TrackContextMessage[any], 0),
		// filter: params.Filter,
		ctx:    c,
		cancel: cancel,
		// pipes:     make([]*pipelines.PipelineDummy, 1),
		// sampleBus: make(chan *media.Sample, 10),
	}

	if err := trackContext.SetFilter(params.Filter); err != nil {
		log.Printf("TRACK | %s unable set filter. Err: %s", trackContext.id, err)
	}

	return trackContext
}

type ActiveTrackContext struct {
	trackContext *TrackContext
	sender       atomic.Pointer[*webrtc.RTPSender]
}

func (a *ActiveTrackContext) LoadSender() *webrtc.RTPSender {
	return *a.sender.Load()
}

func (a *ActiveTrackContext) StoreSender(s *webrtc.RTPSender) {
	a.sender.Store(&s)
}

func (a *ActiveTrackContext) SwitchActiveTrackMedia(pc *webrtc.PeerConnection) error {
	sender := a.LoadSender()
	if sender == nil {
		return ErrSwitchActiveTrackNotFoundSender
	}

	if err := pc.RemoveTrack(sender); err != nil {
		return err
	}

	var err error
	sender = nil

	sender, err = pc.AddTrack(a.trackContext.GetLocalTrack())
	if err != nil {
		return err
	}

	a.StoreSender(sender)
	return nil
}

func NewActiveTrackContext(sender *webrtc.RTPSender, track *TrackContext) *ActiveTrackContext {
	a := &ActiveTrackContext{trackContext: track}
	a.StoreSender(sender)
	return a
}
