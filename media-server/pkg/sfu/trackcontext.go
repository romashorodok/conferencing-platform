package sfu

import (
	"context"

	webrtc "github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type TrackContext struct {
	track *webrtc.TrackLocalStaticSample

	sender *webrtc.RTPSender

	// pipes     []Pipeline
	// sampleBus chan *media.Sample

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (t *TrackContext) WriteSample(sample media.Sample) error {
	return t.track.WriteSample(sample)
}

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
	return t.track.ID()
}

type NewTrackContextParams struct {
	Track  *webrtc.TrackLocalStaticSample
	Sender *webrtc.RTPSender
}

func NewTrackContext(ctx context.Context, params NewTrackContextParams) *TrackContext {
	c, cancel := context.WithCancelCause(ctx)
	return &TrackContext{
		track:  params.Track,
		sender: params.Sender,
		ctx:    c,
		cancel: cancel,
		// pipes:     make([]*pipelines.PipelineDummy, 1),
		// sampleBus: make(chan *media.Sample, 10),
	}
}
