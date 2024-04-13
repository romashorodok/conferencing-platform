package mcu

// #cgo pkg-config: gstreamer-full-1.0
/*
#include <gst/gst.h>
#include <gst/app/gstappsrc.h>
*/
// #cgo pkg-config: media-server-mcu
// #include <main.h>
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

type GstBuffer struct {
	source *C.GstBuffer
}

func BufferNewWrapped(data []byte) (*GstBuffer, error) {
	Cdata := (*C.gchar)(unsafe.Pointer(C.malloc(C.size_t(len(data)))))
	C.bcopy(unsafe.Pointer(&data[0]), unsafe.Pointer(Cdata), C.size_t(len(data)))
	CGstBuffer := C.MCU_gst_buffer_new_wrapped(Cdata, C.gsize(len(data)))
	if CGstBuffer == nil {
		return nil, errors.New("could not allocate and wrap a new GstBuffer")
	}
	return &GstBuffer{
		source: CGstBuffer,
	}, nil
}

func AppSrcPushBuffer(src *GstElement, buf *GstBuffer) error {
	ret := C.gst_app_src_push_buffer((*C.GstAppSrc)(unsafe.Pointer(src.element)), buf.source)
	if FlowReturn(ret) != FlowOK {
		return errors.New("could not push buffer into src")
	}
	return nil
}

type GstSample struct {
	Buff   unsafe.Pointer
	Size   C.int
	Width  uint32
	Height uint32
}

func (s *GstSample) Data() []byte {
	return C.GoBytes(s.Buff, s.Size)
}

func (s *GstSample) Deinit() {
	C.free(s.Buff)
}

func AppSinkPullSample(element *GstElement) (sample *GstSample, err error) {
	CGstSample := C.gst_app_sink_pull_sample((*C.GstAppSink)(unsafe.Pointer(element.element)))
	if CGstSample == nil {
		return nil, errors.New("could not pull a sample from appsink")
	}
	defer C.gst_sample_unref(CGstSample)

	var width, height C.gint
	CCaps := C.gst_sample_get_caps(CGstSample)
	CCStruct := C.gst_caps_get_structure(CCaps, 0)
	C.gst_structure_get_int(CCStruct, (*C.gchar)(unsafe.Pointer(C.CString("width"))), &width)
	C.gst_structure_get_int(CCStruct, (*C.gchar)(unsafe.Pointer(C.CString("height"))), &height)

	CGstBuffer := C.gst_sample_get_buffer(CGstSample)
	if CGstBuffer == nil {
		return nil, errors.New("could not get a sample buffer")
	}

	var CCopy C.gpointer
	var CCopy_size C.gsize
	C.gst_buffer_extract_dup(CGstBuffer, 0, C.gst_buffer_get_size(CGstBuffer), &CCopy,
		&CCopy_size)

	sample = &GstSample{
		Buff:   unsafe.Pointer(CCopy),
		Size:   C.int(CCopy_size),
		Width:  uint32(width),
		Height: uint32(height),
	}

	return sample, nil
}

func SampleUnref(s *GstSample) {
	runtime.GC()
	// C.free(s.CData)
}

func AppSinkIsEOS(element *GstElement) bool {
	Cbool := C.gst_app_sink_is_eos((*C.GstAppSink)(unsafe.Pointer(element.element)))
	if Cbool == 1 {
		return true
	}

	return false
}
