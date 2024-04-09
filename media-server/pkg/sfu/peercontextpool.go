package sfu

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"golang.org/x/sync/errgroup"
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

	if _, exist := s.pool[sub.peerID]; exist {
		return errors.New("Subscriber exist. Remove it first")
	}

	s.pool[sub.peerID] = sub
	return nil
}

func (s *PeerContextPool) Remove(sub *PeerContext) (err error) {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	if sub == nil {
		return
	}

	if s, exist := s.pool[sub.peerID]; exist {
		err = s.Close(ErrPeerConnectionClosed)
	}

	delete(s.pool, sub.peerID)
	return err
}

func (s *PeerContextPool) ForEachAsync(ctx context.Context, f func(*PeerContext) error) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, peer := range s.pool {
		peer := peer
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			return f(peer)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (s *PeerContextPool) TrackDownToPeers(peerOrigin *PeerContext, t *TrackContext) error {
	return s.ForEachAsync(t.ctx, func(peer *PeerContext) error {
		if peer.PeerID() == peerOrigin.PeerID() {
			return nil
		}

		ack := peer.Subscriber.AttachTrack(t)
		if err := <-ack.Result; err != nil {
			return err
		}

		return nil
	})
}

func (s *PeerContextPool) TrackDownStopToPeers(peerOrigin *PeerContext, t *TrackContext) error {
	return s.ForEachAsync(t.ctx, func(peer *PeerContext) error {
		if peer.PeerID() == peerOrigin.PeerID() {
			return nil
		}

		ack := peer.Subscriber.DetachTrack(t)
		if err := <-ack.Result; err != nil {
			return err
		}

		return nil
	})
}

func (s *PeerContextPool) SanitizePeerSenders(peerTarget *PeerContext) error {
	attachedSenders := make(map[string]*webrtc.RTPSender)

	for _, peer := range s.pool {
		for tID, pubTrack := range peer.publishTracks {
			if track, exist := peerTarget.Subscriber.HasTrack(tID); exist {
				sender := track.LoadSender()
				if sender != nil {
					attachedSenders[track.trackContext.ID()] = sender
					continue
				}
			}

			ack := peerTarget.Subscriber.AttachTrack(pubTrack.trackContext)
			if err := <-ack.Result; err != nil {
				return err
			}

			activeTrack, exist := peerTarget.Subscriber.HasTrack(pubTrack.trackContext.ID())
			if !exist {
				return errors.Join(ErrTrackNotFound, fmt.Errorf("track %s must exist", pubTrack.trackContext.ID()))
			}

			attachedSenders[activeTrack.trackContext.ID()] = activeTrack.LoadSender()
		}
	}

	senderReplacerRetry := func() bool {
		for _, sender := range peerTarget.peerConnection.GetSenders() {
			if sender == nil {
				return true
			}

			track := sender.Track()
			if track == nil {
				return true
			}
			tID := track.ID()

			s, exist := attachedSenders[tID]
			if !exist {
				_ = peerTarget.peerConnection.RemoveTrack(sender)
				continue
			}

			if s != sender {
				_ = peerTarget.peerConnection.RemoveTrack(s)
				continue
			}
		}

		return false
	}

	sleep := func() { time.Sleep(time.Millisecond * 100) }

retry:
	for attempt := 0; ; attempt++ {
		log.Println("[Sanitize] attempt", attempt)
		select {
		case <-peerTarget.Done():
			return ErrPeerConnectionClosed
		default:
		}

		if attempt >= 25 {
			sleep()
			log.Println("[Sanitize] retry attempt")
			goto retry
		}

		if !senderReplacerRetry() {
			break
		}
	}

	log.Println("sanitized peer", peerTarget.peerID)
	log.Println("sanitized senders", peerTarget.peerConnection.GetSenders())
	for _, sender := range peerTarget.peerConnection.GetSenders() {
		log.Printf("sanitized track: %s stream: %s", sender.Track().ID(), sender.Track().StreamID())
	}

	_ = peerTarget.Signal.DispatchOffer()

	return nil
}

var _ trackSpreader = (*PeerContextPool)(nil)

func NewPeerContextPool() *PeerContextPool {
	return &PeerContextPool{
		pool: make(map[string]*PeerContext),
	}
}
