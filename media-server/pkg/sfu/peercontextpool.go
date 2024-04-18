package sfu

import (
	"context"
	"errors"
	"log"
	"sync"

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

type UnattachedSender struct {
	track *PublishTrackContext
}

type OptionalSender interface {
	*webrtc.RTPSender | UnattachedSender
}

type OptionalSenderBox[F any] struct {
	value F
}

func (b *OptionalSenderBox[F]) Untype() OptionalSenderBox[any] {
	return OptionalSenderBox[any]{
		value: b.value,
	}
}

func NewOptionalSenderBox[F *webrtc.RTPSender | UnattachedSender](val F) OptionalSenderBox[F] {
	return OptionalSenderBox[F]{
		value: val,
	}
}

func (s *PeerContextPool) PeerPublishingSenders(peerTarget *PeerContext) map[string]OptionalSenderBox[any] {
	pubSenders := make(map[string]OptionalSenderBox[any])

	for _, peer := range s.pool {
		for pubTrackID, pubTrack := range peer.publishTracks {
			if track, exist := peerTarget.Subscriber.HasTrack(pubTrackID); exist {
				sender := track.LoadSender()
				if sender != nil {
					box := NewOptionalSenderBox(sender)
					pubSenders[track.trackContext.ID()] = box.Untype()
					continue
				}
			}

			box := NewOptionalSenderBox(UnattachedSender{
				track: pubTrack,
			})
			pubSenders[pubTrackID] = box.Untype()
		}
	}

	return pubSenders
}

func (s *PeerContextPool) removeUnpublishSenders(peerTarget *PeerContext, pubSenders map[string]OptionalSenderBox[any]) {
	// log.Println(
	// 	"senders:", peerTarget.peerConnection.GetSenders(),
	// 	"pubSenders:", pubSenders,
	// )

	for _, existingSender := range peerTarget.peerConnection.GetSenders() {
		if existingSender == nil {
			continue
		}

		track := existingSender.Track()
		if track == nil {
			continue
		}
		tID := track.ID()

		s, exist := pubSenders[tID]
		if !exist {
			_ = peerTarget.peerConnection.RemoveTrack(existingSender)
			return
		}

		switch sender := s.value.(type) {
		case *webrtc.RTPSender:
			if sender != existingSender {
				_ = peerTarget.peerConnection.RemoveTrack(sender)
				continue
			}
		case UnattachedSender:
			continue
		default:
			log.Println("[removeUnpublishSenders] What is it???", sender)
			panic("[removeUnpublishSenders] Unreachable state")
		}
	}
}

func (s *PeerContextPool) SanitizePeerSenders(peerTarget *PeerContext) error {
	senders := s.PeerPublishingSenders(peerTarget)

	s.removeUnpublishSenders(peerTarget, senders)

	for _, sender := range senders {
		switch s := sender.value.(type) {
		case *webrtc.RTPSender:
			// NOTE: here may be bug if sender not attached but i assume that all ok
			continue
		case UnattachedSender:
			log.Println("Block attached senders")
			t := s.track.trackContext
			ack := peerTarget.Subscriber.AttachTrack(t)
			if err := <-ack.Result; err != nil {
				return err
			}
			log.Println("Unblock attached senders")
		default:
			log.Println("[SanitizePeerSenders] What is it???", s)
			panic("[SanitizePeerSenders] Unreachable state")
		}
	}

	// log.Println("[SanitizePeerSenders] sanitized peer", peerTarget.peerID)
	// log.Println("[SanitizePeerSenders] sanitized senders", peerTarget.peerConnection.GetSenders())

	return peerTarget.Signal.DispatchOffer()
}

var _ trackSpreader = (*PeerContextPool)(nil)

func NewPeerContextPool() *PeerContextPool {
	return &PeerContextPool{
		pool: make(map[string]*PeerContext),
	}
}
