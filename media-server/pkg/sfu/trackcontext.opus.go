package sfu

import (
	// "github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

type TrackContextOpus struct {
	*TrackContext
	sample *samplebuilder.SampleBuilder
}

// func (t *TrackContextOpus) WriteRTP(p *rtp.Packet) error {
// 	t.sample.Push(p)
//
// 	sample := t.sample.Pop()
// 	if sample == nil {
// 		return nil
// 	}
//
// 	return t.passThroughSink(sample)
// }

func NewTrackContextOpus(t *TrackContext) *TrackContextOpus {
	// sample := samplebuilder.New(11, &codecs.OpusPacket{}, t.sample.Codec().ClockRate)
	return &TrackContextOpus{
		TrackContext: t,
		// sample:       sample,
	}
}
