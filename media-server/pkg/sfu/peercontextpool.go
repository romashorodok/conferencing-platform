package sfu

import (
	"errors"
	"sync"
)

type PeerContextPool struct {
	subscriberMu sync.Mutex
	pool         map[string]*PeerContext
}

func (s *PeerContextPool) DispatchOffers() {
	for _, peerContext := range s.pool {
		peerContext.Signal.DispatchOffer()
	}
}

func (s *PeerContextPool) Get() []*PeerContext {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	var result []*PeerContext
	for _, sub := range s.pool {
		result = append(result, sub)
	}

	return result
}

func (s *PeerContextPool) Add(sub *PeerContext) error {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	if _, exist := s.pool[sub.PeerID]; exist {
		return errors.New("Subscriber exist. Remove it first")
	}

	s.pool[sub.PeerID] = sub
	return nil
}

func (s *PeerContextPool) Remove(sub *PeerContext) (err error) {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	if sub == nil {
		return
	}

	if s, exist := s.pool[sub.PeerID]; exist {
		err = s.Close(ErrPeerConnectionClosed)
	}

	delete(s.pool, sub.PeerID)
	return err
}

func NewPeerContextPool() *PeerContextPool {
	return &PeerContextPool{
		pool: make(map[string]*PeerContext),
	}
}
