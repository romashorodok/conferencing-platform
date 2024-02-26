package sfu

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/twcc"
)

type TrackContext struct {
	track   *webrtc.TrackLocalStaticRTP
	sender  *webrtc.RTPSender
	twcc    *twcc.Responder
	twccExt uint8

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (t *TrackContext) WriteRTP(p *rtp.Packet) error {
	go func() {
		if t.twccExt <= 0 {
			return
		}

		ext := p.GetExtension(t.twccExt)
		if ext == nil {
			return
		}

		t.twcc.Push(binary.BigEndian.Uint16(ext[0:2]), time.Now().UnixNano(), p.Marker)
	}()
	return t.track.WriteRTP(p)
}

func (t *TrackContext) Close() (err error) {
	err = t.sender.ReplaceTrack(nil)
	err = t.sender.Stop()
	t.cancel(ErrTrackCancelByUser)
	return
}

func (t *TrackContext) OnCloseAsync(f func()) {
	go func() {
		select {
		case <-t.ctx.Done():
			f()
		}
	}()
}

func (t *TrackContext) OnFeedback(f func(pkts []rtcp.Packet)) {
	t.twcc.OnFeedback(f)
}

type NewTrackContextParams struct {
	Track    *webrtc.TrackLocalStaticRTP
	Sender   *webrtc.RTPSender
	TWCC_EXT uint8
	SSRC     uint32
}

func NewTrackContext(ctx context.Context, params NewTrackContextParams) *TrackContext {
	c, cancel := context.WithCancelCause(ctx)
	return &TrackContext{
		track:   params.Track,
		sender:  params.Sender,
		twcc:    twcc.NewTransportWideCCResponder(params.SSRC),
		twccExt: params.TWCC_EXT,
		ctx:     c,
		cancel:  cancel,
	}
}
