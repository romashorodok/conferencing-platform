package sfu

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/pion/webrtc/v4"
)

type TransceiverPool struct {
	poolMu sync.RWMutex
	count  atomic.Uint32 /* remember 0 index */
	free   map[string]*webrtc.RTPTransceiver
}

func (s *TransceiverPool) Get() (*webrtc.RTPTransceiver, error) {
	s.poolMu.RLock()
	defer s.poolMu.RUnlock()

	log.Printf("transciever pool state %+v", s.free)

	if hasFree := len(s.free); hasFree == 0 {
		return nil, ErrNotFoundTransceiver
	}

	var t *webrtc.RTPTransceiver
	for _, transiv := range s.free {
		if transiv.Direction() != webrtc.RTPTransceiverDirectionInactive {
			panic("[TransceiverPoolGet] Never must be here")
		}
		t = transiv
	}

	delete(s.free, t.Mid())

	return t, nil
}

func (s *TransceiverPool) Release(transiv *webrtc.RTPTransceiver) error {
	s.poolMu.Lock()
	defer s.poolMu.Unlock()

	log.Printf("transciever pool state %+v", s.free)

	if err := transiv.Stop(); err != nil {
		return err
	}

	mid := transiv.Mid()

	s.free[mid] = transiv

	return nil
}

func NewTransceiverPool() *TransceiverPool {
	pool := &TransceiverPool{
		free: make(map[string]*webrtc.RTPTransceiver),
	}
	pool.count.Store(0)
	return pool
}
