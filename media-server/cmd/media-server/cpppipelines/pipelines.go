package cpppipelines

/*
#cgo LDFLAGS: -lstdc++

#cgo pkg-config: pipelines-1.0
#cgo pkg-config: rtpvp8-1.0
#include "pipelines/pipelines.h"
#include "pipelines/rtpvp8/rtpvp8.h"
*/
import "C"

import (
	"unsafe"
)

//export CGO_onSampleBuffer
func CGO_onSampleBuffer(buffer unsafe.Pointer, size C.int, duration C.int) {
	// log.Println("Golang recv", buffer, size, duration)
    C.free(buffer)
}

func GstreamerMainLoopSetup() {
	C.setup()
}
