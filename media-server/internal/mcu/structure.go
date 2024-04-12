package mcu

// #cgo pkg-config: gstreamer-full-1.0
// #include <gst/gst.h>
import "C"

type GstStructure struct {
	C *C.GstStructure
}

