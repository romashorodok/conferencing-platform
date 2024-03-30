package cpppipelines

/*
#include "pipelines/pipelines.h"
#include "pipelines/rtpvp8/rtpvp8.h"
*/
import "C"

import (
	"log"
	"time"
	"unsafe"

	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
)

type RtpVP8 struct {
	pipeline unsafe.Pointer
	trackID  unsafe.Pointer
}

// Sink implements pipelines.Pipeline.
func (p *RtpVP8) Sink(frame []byte, timestamp time.Time, duration time.Duration) error {
	C.write_pipe(p.pipeline, C.CBytes(frame), C.int(len(frame)))
	return nil
}

func (p *RtpVP8) Start() {
	C.start_pipe(p.pipeline)
}

func (p *RtpVP8) Close() {
	C.delete_pipe(p.pipeline)
	C.free(p.pipeline)
}

var _ sfu.Pipeline = (*RtpVP8)(nil)

//export CGO_rtp_vp8_dummy_sample
func CGO_rtp_vp8_dummy_sample(trackID *C.char, buffer unsafe.Pointer, size C.int, duration C.int) {
	defer C.free(buffer)
	// log.Println("on sample", trackID)
	track := sfu.TrackContextRegistry.GetByID(C.GoString(trackID))
	if track == nil {
		log.Printf("Drop sample for track: %s", C.GoString(trackID))
		return
	}

	writer, err := track.GetTrackRemoteWriterSample()
	if err != nil {
		log.Printf("Drop sample for track unable find writer: %s", C.GoString(trackID))
        return
	}

	err = writer.WriteRemote(media.Sample{
		Data: C.GoBytes(buffer, size),
		// NOTE: Relying on a hardcoded duration could lead to the unexpected behavior.
		// But actually it's better than pion handle samples.
		Duration: time.Millisecond,
	})

	// log.Println("result of write", err)

	// track.WriteSample(media.Sample{
	// 	Data: C.GoBytes(buffer, size),
	// 	// NOTE: Relying on a hardcoded duration could lead to the unexpected behavior.
	// 	// But actually it's better than pion handle samples.
	// 	Duration: time.Millisecond,
	// })
}

func NewRtpVP8(trackID string, mimeType string, clockRate uint32) sfu.Pipeline {
	tID := C.CString(trackID)

	// TODO: instead of extern make func callback
	// TODO: check if it's possible, because I cannot use go pointers with c pointers
	pipeline := C.new_pipe_rtp_vp8(tID)

	return &RtpVP8{
		pipeline: pipeline,
		trackID:  unsafe.Pointer(tID),
	}
}
