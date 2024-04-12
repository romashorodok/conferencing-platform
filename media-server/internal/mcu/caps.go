package mcu

// #cgo pkg-config: gstreamer-full-1.0
/*
#include <gst/gst.h>
#include <gst/gstcaps.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

type GstCaps struct {
	caps *C.GstCaps
}

func NewRtpVP8Caps(clockRate uint32) string {
	return fmt.Sprintf("application/x-rtp, media=(string)video, payload=(int)96, clock-rate=(int)%d, encoding-name=(string)VP8-DRAFT-IETF-01", clockRate)
}

func CapsFromString(caps string) *GstCaps {
	c := (*C.gchar)(unsafe.Pointer(C.CString(caps)))
	defer C.g_free(C.gpointer(unsafe.Pointer(c)))
	CCaps := C.gst_caps_from_string(c)
	gstCaps := &GstCaps{caps: CCaps}

	runtime.SetFinalizer(gstCaps, func(gstCaps *GstCaps) {
		C.gst_caps_unref(gstCaps.caps)
	})

	return gstCaps
}
