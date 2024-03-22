package sfu

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

const RTP_VP8_DUMMY = "rtp-vp8-dummy"

type MimeType string

func (m *MimeType) String() string {
	return string(*m)
}

var (
	MIME_TYPE_VIDEO = MimeType("video")
	MIME_TYPE_AUDIO = MimeType("audio")
)

type Filter struct {
	Name      string
	MimeTypes []MimeType
	// TODO: refactor `Allocator` to be here
	Allocator Allocator
}

// Filter("none")
// Filter(RTP_VP8_DUMMY)
var (
	FILTER_NONE = &Filter{
		Name: "none",
		MimeTypes: []MimeType{
			MIME_TYPE_VIDEO,
			MIME_TYPE_AUDIO,
		},
		Allocator: nil,
	}

	FILTER_RTP_VP8_DUMMY = &Filter{
		Name: RTP_VP8_DUMMY,
		MimeTypes: []MimeType{
			MIME_TYPE_VIDEO,
		},
		Allocator: nil,
	}
)

func (f *Filter) GetName() string {
	return f.Name
}

type Pipeline interface {
	Sink(frame []byte, timestamp time.Time, duration time.Duration) error

	Start()
	Close()
}

type Allocator = func(trackID string, mimeType string, clockRate uint32) Pipeline

type AllocatorsContext struct {
	allocators map[*Filter]Allocator
}

func (ctx *AllocatorsContext) Register(name *Filter, alloc Allocator) {
	if _, ok := ctx.allocators[name]; ok {
		log.Panic("Invalid allocator name")
		os.Exit(1)
	}
	ctx.allocators[name] = alloc
}

func (ctx *AllocatorsContext) Filter(name string) (*Filter, error) {
	if name == FILTER_NONE.GetName() {
		return FILTER_NONE, nil
	}

	for filter := range ctx.allocators {
		if filter.GetName() == name {
			return filter, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("unable find %s filter", name))
}

type AllocateParams struct {
	TrackID   string
	Filter    *Filter
	MimeType  string
	ClockRate uint32
}

func (ctx *AllocatorsContext) Allocate(params *AllocateParams) (Pipeline, error) {
	alloc, ok := ctx.allocators[params.Filter]
	if !ok {
		return nil, ErrInvalidPipelineAllocatorName
	}

	return alloc(params.TrackID, params.MimeType, params.ClockRate), nil
}

func (ctx *AllocatorsContext) Filters() []*Filter {
	result := []*Filter{FILTER_NONE}
	for filter := range ctx.allocators {
		result = append(result, filter)
	}
	return result
}

func NewAllocatorsContext() *AllocatorsContext {
	return &AllocatorsContext{
		make(map[*Filter]func(string, string, uint32) Pipeline),
	}
}
