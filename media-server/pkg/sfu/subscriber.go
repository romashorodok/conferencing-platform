package sfu

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v4"
)

type watchTrackAck struct {
	TrackContext *TrackContext
	Result       chan error
}

func NewWatchTrackAck(t *TrackContext) watchTrackAck {
	return watchTrackAck{
		TrackContext: t,
		Result:       make(chan error),
	}
}

type Subscriber struct {
	webrtc         *webrtc.API
	peerConnection *webrtc.PeerConnection

	peerId string
	sid    string

	tracks   sync.Map
	tracksMu sync.Mutex

	handleAttachTrackMu sync.Mutex
	busAttachTrack      chan watchTrackAck
	handleDetachTrackMu sync.Mutex
	busDetachTrack      chan watchTrackAck

	observers   atomic.Pointer[[]chan SubscriberMessage[any]]
	observersMu sync.Mutex

	transceiverPool  *TransceiverPool

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (s *Subscriber) MapStoreTrack(tID string, t *ActiveTrackContext) {
	s.tracks.Store(tID, t)
}

func (s *Subscriber) MapDeleteTrack(tID string) {
	s.tracks.Delete(tID)
}

func (s *Subscriber) MapExistTrack(tID string) (any, bool) {
	val, exist := s.tracks.Load(tID)
	return val, exist
}

func (s *Subscriber) MapForEachTrack(f func(key, value any) bool) {
	s.tracks.Range(f)
}

func (s *Subscriber) HandleTrackAttach() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case ack := <-s.busAttachTrack:
			s.handleAttachTrackMu.Lock()
			t := ack.TrackContext

			track, found := s.HasTrack(t.ID())
			if !found {
				log.Printf("track %s not exists", t.ID())

				var err error
				var transiv *webrtc.RTPTransceiver

				transiv, err = s.transceiverPool.Get()
				if err != nil {
					switch err {
					case ErrNotFoundTransceiver:
						if transiv, err = s.peerConnection.AddTransceiverFromKind(t.codecKind, webrtc.RTPTransceiverInit{
							Direction: webrtc.RTPTransceiverDirectionSendonly,
						}); err != nil {
							log.Printf("Ignore %s track for sub %s. Unable create transiver track. Err:%s", t.ID(), s.peerId, err)
							ack.Result <- err
							close(ack.Result)
							s.handleAttachTrackMu.Unlock()
							continue
						}
					default:
						log.Printf("Ignore %s track for sub %s. Unable create transiver track. Err:%s", t.ID(), s.peerId, err)
						ack.Result <- err
						close(ack.Result)
						s.handleAttachTrackMu.Unlock()
						continue
					}
				}

				var sender *webrtc.RTPSender
				var trackSender webrtc.TrackLocal

				switch t.codecKind {
				case webrtc.RTPCodecTypeAudio:
					if s.peerId == t.SourcePeerID {
						log.Printf("[Track %s] found loopback audio send stub for %s",
							t.ID(),
							t.SourcePeerID,
						)
						if trackSender, err = webrtc.NewTrackLocalStaticSample(
							t.codecParams.RTPCodecCapability,
							t.ID(),
							t.streamID,
						); err != nil {
							break
						}
						sender, err = s.webrtc.NewRTPSender(trackSender, s.peerConnection.SCTP().Transport())
						break
					}

					fallthrough
				default:
					trackSender = t.GetLocalTrack()
					sender, err = s.webrtc.NewRTPSender(trackSender, s.peerConnection.SCTP().Transport())
				}

				if err != nil {
					ack.Result <- err
					close(ack.Result)
					s.handleAttachTrackMu.Unlock()
					continue
				}

				if err = transiv.SetSender(sender, trackSender); err != nil {
					ack.Result <- err
					close(ack.Result)
					s.handleAttachTrackMu.Unlock()
				}

				track = NewActiveTrackContext(transiv, transiv.Sender(), t)
				s.MapStoreTrack(t.ID(), track)

				ack.Result <- nil
				close(ack.Result)
				s.dispatch(NewSubscriberMessage(SubscriberTrackAttached{
					track: track,
				}))
			} else {
				log.Printf("track %s already exists", track.trackContext.ID())

				ack.Result <- nil
				close(ack.Result)
			}

			s.handleAttachTrackMu.Unlock()
		}
	}
}

func (s *Subscriber) HandleTrackDetach() {
	for {
		select {
		case <-s.ctx.Done():
			log.Println("peer context stop detach")
			return
		case ack := <-s.busDetachTrack:
			s.handleDetachTrackMu.Lock()
			t := ack.TrackContext
			// log.Println("detach track | search for", t.ID())

			track, found := s.HasTrack(t.ID())
			if !found {
				// log.Printf("detach track | not found %s track. Ignoring...", t.ID())
				ack.Result <- errors.Join(ErrWatchTrackDetachNotFound, fmt.Errorf("PeerID: %s TrackID:%s", s.peerId, t.ID()))
				close(ack.Result)
				return
			}

			t = track.trackContext
			s.MapDeleteTrack(t.ID())

			sender := track.LoadSender()
			if sender == nil {
				// NOTE: If this called, something goes too wrong
				ack.Result <- errors.Join(ErrWatchTrackDetachSenderNotFound, fmt.Errorf("Track not attached. PeerID: %s TrackID:%s", s.peerId, t.ID()))
				close(ack.Result)
				return
			}

			err := s.peerConnection.RemoveTrack(sender)
			if err != nil {
				ack.Result <- err
			} else {
				ack.Result <- nil
			}

			tranciv := track.LoadTransiver()
			err = s.transceiverPool.Release(tranciv)
			if err != nil {
				log.Println("Unable release transiver")
			}

			s.dispatch(NewSubscriberMessage(SubscriberTrackDetached{
				track: track,
			}))

			close(ack.Result)
			s.handleDetachTrackMu.Unlock()
		}
	}
}

func (s *Subscriber) Track(streamID string, t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) *TrackContext {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	id := t.ID()
	if id == "" {
		id = uuid.NewString()
	}

	return NewTrackContext(s.ctx, NewTrackContextParams{
		SourcePeerID: s.peerId,
		ID:           id,
		StreamID:     streamID,
		RID:          t.RID(),
		SSRC:         t.SSRC(),
		PayloadType:  t.PayloadType(),

		CodecParams: t.Codec(),
		Kind:        t.Kind(),

		PeerConnection: s.peerConnection,
		API:            s.webrtc,
	})
}

func (s *Subscriber) AttachTrack(t *TrackContext) watchTrackAck {
	ack := NewWatchTrackAck(t)
	s.busAttachTrack <- ack
	return ack
}

func (s *Subscriber) DetachTrack(t *TrackContext) watchTrackAck {
	ack := NewWatchTrackAck(t)
	s.busDetachTrack <- ack
	return ack
}

var EmptyActiveTrackContext = &ActiveTrackContext{}

func (s *Subscriber) HasTrack(trackID string) (*ActiveTrackContext, bool) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	track, exist := s.MapExistTrack(trackID)
	if !exist || track == nil {
		return EmptyActiveTrackContext, exist
	}
	return track.(*ActiveTrackContext), exist
}

func (s *Subscriber) DeleteTrack(t *ActiveTrackContext) error {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	id := t.trackContext.ID()
	track, exist := s.MapExistTrack(id)
	if !exist || track == nil {
		return fmt.Errorf("Track not exist. Unable delete %s", id)
	}
	s.MapDeleteTrack(id)

	err := s.transceiverPool.Release(t.LoadTransiver())
	if err != nil {
		log.Println("[SubscriberDeleteTrack] Unable release transciever")
	}

	return s.peerConnection.RemoveTrack(track.(*ActiveTrackContext).LoadSender())
}

func (s *Subscriber) Close() (err error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	s.MapForEachTrack(func(key, value any) bool {
		id := key.(string)
		active := value.(*ActiveTrackContext)

		_ = s.peerConnection.RemoveTrack(active.LoadSender())
		_ = active.trackContext.Close()

		s.MapDeleteTrack(id)
		return true
	})

	return err
}

func (s *Subscriber) Done() <-chan struct{} {
	return s.ctx.Done()
}
