package sfu

import (
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

type TrackContextVp8 struct {
	*TrackContext
	sample *samplebuilder.SampleBuilder
}

// func (t *TrackContextVp8) WriteRTP(p *rtp.Packet) error {
// 	t.sample.Push(p)
//
// 	sample := t.sample.Pop()
// 	if sample == nil {
// 		return nil
// 	}
//
// 	return t.passThroughSink(sample)
// }

func NewTrackContextVp8(t *TrackContext) *TrackContextVp8 {
	// sample := samplebuilder.New(10, &codecs.VP8Packet{}, t.sample.Codec().ClockRate)

	return &TrackContextVp8{
		TrackContext: t,
		// sample:       sample,
	}
}
