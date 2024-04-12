package mcu

// #cgo pkg-config: gstreamer-full-1.0
// #include <gst/gst.h>
// #cgo pkg-config: media-server-mcu
// #include <main.h>
import "C"

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"unsafe"
)

type GstElement struct {
	element *C.GstElement
}

func ElementFactoryMake(factoryName string, name string) (*GstElement, error) {
	var pName *C.gchar

	pFactoryName := (*C.gchar)(unsafe.Pointer(C.CString(factoryName)))
	defer C.g_free(C.gpointer(unsafe.Pointer(pFactoryName)))
	if name == "" {
		pName = nil
	} else {
		pName = (*C.gchar)(unsafe.Pointer(C.CString(name)))
		defer C.g_free(C.gpointer(unsafe.Pointer(pName)))
	}

	elem := C.gst_element_factory_make(pFactoryName, pName)

	if elem == nil {
		err := errors.New(fmt.Sprintf("could not create a GStreamer element factoryName %s, name %s", factoryName, name))
		return nil, err
	}

	return &GstElement{element: elem}, nil
}

func PipelineNew(name string) (*GstElement, error) {
	var pName *C.gchar

	if name == "" {
		pName = nil
	} else {
		pName = (*C.gchar)(unsafe.Pointer(C.CString(name)))
		defer C.g_free(C.gpointer(unsafe.Pointer(pName)))
	}

	gstElem := C.gst_pipeline_new(pName)
	if gstElem == nil {
		err := errors.New(fmt.Sprintf("could not create a Gstreamer pipeline name %s", name))
		return nil, err
	}

	elem := &GstElement{element: gstElem}

	runtime.SetFinalizer(elem, func(e *GstElement) {
		fmt.Printf("CLEANING PIPELINE")
		C.gst_object_unref(C.gpointer(unsafe.Pointer(elem.element)))
	})

	return elem, nil
}

func ObjectSet(elem *GstElement, pName string, pValue any) {
	CpName := (*C.gchar)(unsafe.Pointer(C.CString(pName)))
	defer C.g_free(C.gpointer(unsafe.Pointer(CpName)))

	switch pValue.(type) {
	case string:
		str := (*C.gchar)(unsafe.Pointer(C.CString(pValue.(string))))
		defer C.g_free(C.gpointer(unsafe.Pointer(str)))

		C.MCU_gst_elem_set_string(elem.element, CpName, str)
	case int:
		C.MCU_gst_elem_set_int(elem.element, CpName, C.gint(pValue.(int)))
	case uint32:
		C.MCU_gst_elem_set_uint(elem.element, CpName, C.guint(pValue.(uint32)))
	case bool:
		var value int
		if pValue.(bool) == true {
			value = 1
		} else {
			value = 0
		}
		C.MCU_gst_elem_set_bool(elem.element, CpName, C.gboolean(value))
	case *GstCaps:
		caps := pValue.(*GstCaps)
		C.MCU_gst_elem_set_caps(elem.element, CpName, caps.caps)
	case *GstStructure:
		structure := pValue.(*GstStructure)
		C.MCU_gst_elem_set_structure(elem.element, CpName, structure.C)
	}

	return
}

type FlowReturn int32

const (
	FlowOK FlowReturn = C.GST_FLOW_OK

	FlowNotLinked FlowReturn = C.GST_FLOW_NOT_LINKED
	FlowFlushing  FlowReturn = C.GST_FLOW_FLUSHING

	FlowEOS           FlowReturn = C.GST_FLOW_EOS
	FlowNotNegotiated FlowReturn = C.GST_FLOW_NOT_NEGOTIATED
	FlowError         FlowReturn = C.GST_FLOW_ERROR
	FlowNotSupported  FlowReturn = C.GST_FLOW_NOT_SUPPORTED

	FlowCustomError   FlowReturn = C.GST_FLOW_CUSTOM_ERROR
	FlowCustomError_1 FlowReturn = C.GST_FLOW_CUSTOM_ERROR_1
	FlowCustomError_2 FlowReturn = C.GST_FLOW_CUSTOM_ERROR_2

	FlowCustomSuccess  FlowReturn = C.GST_FLOW_CUSTOM_SUCCESS
	FlowCustomSuccess1 FlowReturn = C.GST_FLOW_CUSTOM_SUCCESS_1
	FlowCustomSuccess2 FlowReturn = C.GST_FLOW_CUSTOM_SUCCESS_2
)

type StateChangeReturn uint32

const (
	StateChangeReturnFailure   StateChangeReturn = C.GST_STATE_CHANGE_FAILURE
	StateChangeReturnSuccess   StateChangeReturn = C.GST_STATE_CHANGE_SUCCESS
	StateChangeReturnAsync     StateChangeReturn = C.GST_STATE_CHANGE_ASYNC
	StateChangeReturnNoPreroll StateChangeReturn = C.GST_STATE_CHANGE_NO_PREROLL
)

type StateOptions int

const (
	StateVoidPending StateOptions = C.GST_STATE_VOID_PENDING
	StateNull        StateOptions = C.GST_STATE_NULL
	StateReady       StateOptions = C.GST_STATE_READY
	StatePaused      StateOptions = C.GST_STATE_PAUSED
	StatePlaying     StateOptions = C.GST_STATE_PLAYING
)

func ElementSetState(element *GstElement, state StateOptions) StateChangeReturn {
	log.Println(element, state)
	return StateChangeReturn(C.gst_element_set_state(element.element, C.GstState(state)))
}

func ElementLink(src, dest *GstElement) bool {
	res := C.gst_element_link(src.element, dest.element)
	if res == C.TRUE {
		return true
	}
	return false
}

func BinAddMany(p *GstElement, elements ...*GstElement) {
	for _, e := range elements {
		if e != nil {
			C.MCU_gst_bin_add(p.element, e.element)
		}
	}
	return
}
