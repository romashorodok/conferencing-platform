package sfu

import "sync"

type TrackContextPool struct {
	trackMu sync.Mutex
	pool    map[string]*TrackContext
}

func (s *TrackContextPool) Get() []*TrackContext {
	var result []*TrackContext
	for _, t := range s.pool {
		result = append(result, t)
	}
	return result
}

func (s *TrackContextPool) GetByID(id string) *TrackContext {
	return s.pool[id]
}

func (s *TrackContextPool) Add(t *TrackContext) error {
	s.trackMu.Lock()
	defer s.trackMu.Unlock()

	if _, exist := s.pool[t.ID()]; exist {
		return ErrTrackAlreadyExists
	}

	s.pool[t.ID()] = t
	return nil
}

func (s *TrackContextPool) Remove(t *TrackContext) (err error) {
	s.trackMu.Lock()
	defer s.trackMu.Unlock()

	if t == nil {
		return
	}

	delete(s.pool, t.ID())
	return err
}

var TrackContextRegistry = &TrackContextPool{
	pool: make(map[string]*TrackContext),
}
