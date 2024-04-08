package sfu

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	webrtc "github.com/pion/webrtc/v4"
)

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type Signal struct {
	signalMu sync.Mutex
	conn     WebsocketWriter
	agent    ICEAgent
}

func (s *Signal) OnCandidate(data []byte) error {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(data, &candidate); err != nil {
		return err
	}
	return s.agent.SetCandidate(candidate)
}

func (s *Signal) OnAnswer(data []byte) error {
	var answer webrtc.SessionDescription
	if err := json.Unmarshal(data, &answer); err != nil {
		return err
	}
	return s.agent.SetAnswer(answer)
}

func (s *Signal) DispatchOffer() error {
	s.signalMu.Lock()
	defer s.signalMu.Unlock()

	sleep := func() { time.Sleep(time.Millisecond * 100) }

	offerWriter := func(offer string) (bool, error) {
		if err := s.conn.WriteJSON(&websocketMessage{
			Event: "offer",
			Data:  offer,
		}); err != nil {
			return false, err
		}
		return true, nil
	}

	for attempt := 0; ; attempt++ {
		// log.Println("[Signal attempt] attempt", attempt)
		offer, err := s.agent.Offer()

		if attempt >= 25 {
			go func() {
				sleep()
				s.DispatchOffer()
			}()
			return ErrSignalRetry
		}

		switch {
		case errors.Is(err, ErrPeerConnectionClosed):
			return err
		case err != nil:
			log.Printf("[Signal SDP] %s", err)
			sleep()
			continue
		default:
		}

		success, err := offerWriter(offer)
		if errors.Is(err, websocket.ErrCloseSent) {
			return err
		}

		if success {
			break
		}
	}
	return nil
}

type WebsocketWriter interface {
	WriteJSON(val any) error
	ReadJSON(val any) error
	Close() error
}

type ICEAgent interface {
	Offer() (offer string, err error)
	SetAnswer(desc webrtc.SessionDescription) error
	SetCandidate(candidate webrtc.ICECandidateInit) error
}

func newSignal(conn WebsocketWriter, agent ICEAgent) *Signal {
	signal := &Signal{
		conn:  conn,
		agent: agent,
	}
	return signal
}
