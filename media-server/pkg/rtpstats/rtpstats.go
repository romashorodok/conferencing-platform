package rtpstats

import (
	"github.com/pion/interceptor/pkg/stats"
)

type RtpStats struct {
	getter stats.Getter
}

func (rStat *RtpStats) GetGetter() stats.Getter {
	return rStat.getter
}

func NewRtpStats(getter stats.Getter) *RtpStats {
	return &RtpStats{getter}
}
