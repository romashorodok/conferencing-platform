package sfu

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/pion/rtp"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type TrackWriterRTP interface {
	WriteRTP(*rtp.Packet) error
}

type TrackRemoteWriterSample interface {
	WriteRemote(sample media.Sample) error
}

type TrackMediaEngine interface {
	TrackWriterRTP
	TrackRemoteWriterSample
	SetPipeline(pipe Pipeline) error
	GetLocalTrack() webrtc.TrackLocal
}

var ErrUnsupportedCaps = errors.New("")

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

var _ TrackMediaEngine = (*TrackMediaEngineRtp)(nil)

func NewTrackMediaEngineRtp(rtp *webrtc.TrackLocalStaticRTP) *TrackMediaEngineRtp {
	return &TrackMediaEngineRtp{
		rtp: rtp,
	}
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

var _ TrackMediaEngine = (*TrackMediaEngineSample)(nil)

func NewTrackMediaEngineSample(sample *webrtc.TrackLocalStaticSample) *TrackMediaEngineSample {
	return &TrackMediaEngineSample{
		sample: sample,
	}
}

type TrackContext struct {
	id string

	media  TrackMediaEngine
	rtp    *webrtc.TrackLocalStaticRTP
	sample *webrtc.TrackLocalStaticSample

	sender           *webrtc.RTPSender
	filter           *Filter
	pipeAllocContext *AllocatorsContext

	// pipes     []Pipeline
	// sampleBus chan *media.Sample

	ctx    context.Context
	cancel context.CancelCauseFunc
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
	err = t.sender.ReplaceTrack(nil)
	err = t.sender.Stop()
	t.cancel(ErrTrackCancelByUser)
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

func (t *TrackContext) ID() string {
	return t.id
}

func (t *TrackContext) SetFilter(filter *Filter) error {
	// if t.rtp.Kind() != t.sample.Kind() {
	// 	log.Panicf("different track mime type on context. RTP: %s SAMPLE: %s", t.rtp.Kind(), t.sample.Kind())
	// 	os.Exit(-1)
	// }

	if t.rtp.Kind() != t.sender.Track().Kind() {
		return errors.New("not allowed replace mime type of the track. Create a new one instead")
	}

	var found bool
	switch kind := t.rtp.Kind(); kind {
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

	var media TrackMediaEngine
	switch filter {
	case FILTER_NONE:
		media = NewTrackMediaEngineRtp(t.rtp)
	case FILTER_RTP_VP8_DUMMY:
		media = NewTrackMediaEngineSample(t.sample)

		caps := t.sample.Codec()
		log.Printf("%+v", t.pipeAllocContext)
		pipe, err := t.pipeAllocContext.Allocate(&AllocateParams{
			TrackID:   t.ID(),
			Filter:    FILTER_RTP_VP8_DUMMY,
			MimeType:  caps.MimeType,
			ClockRate: caps.ClockRate,
		})
        pipe.Start()

		if err != nil {
			return err
		}
		log.Printf("pipe: %+v", pipe)
		_ = media.SetPipeline(pipe)

	default:
		media = NewTrackMediaEngineRtp(t.rtp)
	}

	if err := t.sender.ReplaceTrack(media.GetLocalTrack()); err != nil {
		return err
	}
    log.Printf("replace track %+v", media)

	t.filter = filter
	t.media = media
	return nil
}

func (t *TrackContext) Filter() *Filter {
	return t.filter
}

type NewTrackContextParams struct {
	ID          string
	TrackSample *webrtc.TrackLocalStaticSample
	TrackRtp    *webrtc.TrackLocalStaticRTP
	// TODO: add alloc to filter
	PipeAllocContext *AllocatorsContext
	Sender           *webrtc.RTPSender
	Filter           *Filter
}

// TODO: creating of RTPSender must be here not on sub side
func NewTrackContext(ctx context.Context, params NewTrackContextParams) *TrackContext {
	if params.Filter == nil {
		params.Filter = FILTER_NONE
	}

	c, cancel := context.WithCancelCause(ctx)
	trackContext := &TrackContext{
		id:     params.ID,
		rtp:    params.TrackRtp,
		sample: params.TrackSample,
		// track:  params.Track,
		sender:           params.Sender,
		pipeAllocContext: params.PipeAllocContext,
		// filter: params.Filter,
		ctx:    c,
		cancel: cancel,
		// pipes:     make([]*pipelines.PipelineDummy, 1),
		// sampleBus: make(chan *media.Sample, 10),
	}

	if err := trackContext.SetFilter(params.Filter); err != nil {
		log.Printf("TRACK | %s unable set filter", trackContext.id)
	}

	return trackContext
}
