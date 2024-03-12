package cpppipelines

/*
#include "pipelines/pipelines.h"
#include "pipelines/rtpvp8/rtpvp8.h"
*/
import "C"

import (
	"unsafe"

	"github.com/google/uuid"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/pipelines"
)

type RtpVP8 struct {
	pipeline unsafe.Pointer
	id       string
}

func (p *RtpVP8) Start() {
	C.start_pipe(p.pipeline)
}

func (p *RtpVP8) Close() {
	C.delete_pipe(p.pipeline)
	C.free(p.pipeline)
}

func (p *RtpVP8) Write(data []byte) {
	C.write_pipe(p.pipeline, C.CBytes(data), C.int(len(data)))
}

var _ pipelines.Pipeline = (*RtpVP8)(nil)

func NewRtpVP8() pipelines.Pipeline {
	id := uuid.NewString()

	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))

	pipeline := C.new_pipe_rtp_vp8(cID)

	return &RtpVP8{
		pipeline: pipeline,
		id:       id,
	}
}
