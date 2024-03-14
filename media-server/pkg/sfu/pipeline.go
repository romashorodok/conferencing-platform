package sfu

import (
	"log"
	"os"
	"time"
)

const RTP_VP8_DUMMY = "rtp-vp8-dummy"

type Pipeline interface {
	Sink(frame []byte, timestamp time.Time, duration time.Duration) error

	Start()
	Close()
}

type Allocator = func(trackID string, mimeType string, clockRate uint32) Pipeline

type AllocatorsContext struct {
	allocators map[string]Allocator
}

func (ctx *AllocatorsContext) Register(name string, alloc Allocator) {
	if _, ok := ctx.allocators[name]; ok {
		log.Panic("Invalid allocator name")
		os.Exit(1)
	}
	ctx.allocators[name] = alloc
}

type AllocateParams struct {
	TrackID   string
	PipeName  string
	MimeType  string
	ClockRate uint32
}

func (ctx *AllocatorsContext) Allocate(params *AllocateParams) (Pipeline, error) {
	alloc, ok := ctx.allocators[params.PipeName]
	if !ok {
		return nil, ErrInvalidPipelineAllocatorName
	}

	return alloc(params.TrackID, params.MimeType, params.ClockRate), nil
}

func NewAllocatorsContext() *AllocatorsContext {
	return &AllocatorsContext{
		make(map[string]func(string, string, uint32) Pipeline),
	}
}
