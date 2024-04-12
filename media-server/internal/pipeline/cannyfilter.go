package pipeline

import (
	"log"
	"time"

	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/romashorodok/conferencing-platform/media-server/internal/mcu"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
)

type CannyFilter struct {
	caps *mcu.GstCaps

	pipe                 *mcu.GstElement
	appSrc               *mcu.GstElement
	queueRtpJitterBuffer *mcu.GstElement
	rtpJitterBuffer      *mcu.GstElement
	queueRtpVP8depay     *mcu.GstElement
	rtpVP8depay          *mcu.GstElement
	appSink              *mcu.GstElement

	track *sfu.TrackContext
}

func (c *CannyFilter) Close() error {
	panic("unimplemented")
}

func (c *CannyFilter) Sink(frame []byte, timestamp time.Time, duration time.Duration) error {
	buf, err := mcu.BufferNewWrapped(frame)
	if err != nil {
		return err
	}

	res := mcu.AppSrcPushBuffer(c.appSrc, buf)
	if res != nil {
		log.Println("Write err", res)
	}
	return res
}

func (c *CannyFilter) Start() error {
	go c.handleSample()
	return nil
}

var _ sfu.Pipeline = (*CannyFilter)(nil)

func (c *CannyFilter) handleSample() {
	var gstSample *mcu.GstSample
	var err error

	e := c.appSink
	for {
		gstSample, err = mcu.AppSinkPullSample(e)
		if err != nil {
			log.Println(err)
			if mcu.AppSinkIsEOS(e) == true {
				log.Println("EOS")
				continue
			} else {
				log.Println("could not get sample from sink")
				continue
			}
		}

		w, err := c.track.GetTrackRemoteWriterSample()
		if err != nil {
			log.Println("writer empty", err)
			continue
		}

		err = w.WriteRemote(media.Sample{
			Data:     gstSample.Data,
			Duration: time.Millisecond,
		})
		if err != nil {
			log.Println("write track remote", err)
		}
	}
}

// TODO: use just function
func (c *CannyFilter) New(t *sfu.TrackContext) (sfu.Pipeline, error) {
	var filter CannyFilter
	filter.track = t

	pipe, err := mcu.PipelineNew("")
	if err != nil {
		return nil, err
	}
	filter.pipe = pipe

	src, err := mcu.ElementFactoryMake("appsrc", "")
	if err != nil {
		return nil, err
	}
	filter.caps = mcu.CapsFromString(mcu.NewRtpVP8Caps(t.GetClockRate()))
	filter.appSrc = src

	mcu.ObjectSet(src, "caps", filter.caps)
	mcu.ObjectSet(src, "format", 3)
	mcu.ObjectSet(src, "is-live", true)

	queueRtpJitterBuffer, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueRtpJitterBuffer = queueRtpJitterBuffer

	rtpJitterBuffer, err := mcu.ElementFactoryMake("rtpjitterbuffer", "")
	if err != nil {
		return nil, err
	}
	filter.rtpJitterBuffer = rtpJitterBuffer

	queueRtpVP8depay, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueRtpVP8depay = queueRtpVP8depay

	rtpVP8depay, err := mcu.ElementFactoryMake("rtpvp8depay", "")
	if err != nil {
		return nil, err
	}
	filter.rtpVP8depay = rtpVP8depay

	sink, err := mcu.ElementFactoryMake("appsink", "")
	if err != nil {
		return nil, err
	}
	filter.appSink = sink
	mcu.ObjectSet(sink, "sync", false)
	mcu.ObjectSet(sink, "drop", true)

	mcu.BinAddMany(pipe, src, queueRtpJitterBuffer, rtpJitterBuffer, queueRtpVP8depay, rtpVP8depay, sink)
	succes := mcu.ElementLink(src, queueRtpJitterBuffer)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(queueRtpJitterBuffer, rtpJitterBuffer)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(rtpJitterBuffer, queueRtpVP8depay)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(queueRtpVP8depay, rtpVP8depay)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(rtpVP8depay, sink)
	if !succes {
		panic("unable link GstElement")
	}

	state := mcu.ElementSetState(pipe, mcu.StatePlaying)
	if state != mcu.StateChangeReturnAsync {
		panic("invalid state")
		// return errors.New("could not state canny filter pipeline")
	}

	return &filter, nil
}
