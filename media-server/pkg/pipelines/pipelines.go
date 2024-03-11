package pipelines

import (
	"errors"
	"log"
	"os"
)

type Pipeline interface {
	Start()
	Write([]byte)
	Close()
}

type Allocator = func() Pipeline

const RTP_VP8_BASE = "rtp-vp8-base"

var ErrInvalidPipelineAllocatorName = errors.New("Invalid pipeline allocator name")

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

func (ctx *AllocatorsContext) Allocate(name string) (Pipeline, error) {
	alloc, ok := ctx.allocators[name]
	if !ok {
		return nil, ErrInvalidPipelineAllocatorName
	}
	return alloc(), nil
}

func NewAllocatorsContext() *AllocatorsContext {
	return &AllocatorsContext{
		allocators: make(map[string]func() Pipeline),
	}
}
