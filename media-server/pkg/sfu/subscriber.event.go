package sfu

type SubscriberTrackAttached struct {
	track *ActiveTrackContext
}

func (s *SubscriberTrackAttached) ActiveTrack() *ActiveTrackContext {
	return s.track
}

type SubscriberTrackDetached struct{
	track *ActiveTrackContext
}

func (s *SubscriberTrackDetached) ActiveTrack() *ActiveTrackContext {
	return s.track
}

type SubscriberEvent interface {
	SubscriberTrackAttached | SubscriberTrackDetached
}

func (p *Subscriber) Observer() <-chan SubscriberMessage[any] {
	p.observersMu.Lock()
	defer p.observersMu.Unlock()

	ch := make(chan SubscriberMessage[any])
	observers := append(p.observersLoad(), ch)
	p.observers.Store(&observers)
	return ch
}

func (p *Subscriber) ObserverUnref(obs <-chan SubscriberMessage[any]) {
	p.observersMu.Lock()
	defer p.observersMu.Unlock()

	observers := p.observersLoad()

	for i, observer := range observers {
		if obs == observer {
			close(observer)
			observers = append(observers[:i], observers[i+1:]...)
			p.observers.Store(&observers)
			return
		}
	}
}

func (p *Subscriber) observersLoad() []chan SubscriberMessage[any] {
	var result []chan SubscriberMessage[any]
	if ptr := p.observers.Load(); ptr != nil {
		result = append(result, *ptr...)
	}
	return result
}

func (p *Subscriber) dispatch(msg SubscriberMessage[any]) {
	p.observersMu.Lock()
	defer p.observersMu.Unlock()

	for _, ch := range p.observersLoad() {
		go func(ch chan SubscriberMessage[any]) {
			select {
			case <-p.ctx.Done():
				return
			case ch <- msg:
				return
			}
		}(ch)
	}
}

type SubscriberMessage[F any] struct {
	value F
}

func (m *SubscriberMessage[F]) Unbox() F {
	return m.value
}

func NewSubscriberMessage[F SubscriberEvent](evt F) SubscriberMessage[any] {
	return SubscriberMessage[any]{
		value: evt,
	}
}
