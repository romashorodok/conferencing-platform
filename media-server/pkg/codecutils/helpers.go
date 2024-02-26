package codecutils

import (
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
)

// I-frame - completed picture, can be decoded
func IsVP8IKeyFrame(packet *rtp.Packet) bool {
	var vp8 codecs.VP8Packet
	_, err := vp8.Unmarshal(packet.Payload)

	if err != nil || len(vp8.Payload) < 1 {
		return false
	}

	if vp8.S != 0 && vp8.PID == 0 && (vp8.Payload[0]&0x1) == 0 {
		return true
	}
	return false
}
