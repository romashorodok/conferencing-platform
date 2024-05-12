package pipeline

import (
	"errors"
	"log"
	"time"

	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/romashorodok/conferencing-platform/media-server/internal/mcu"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
)

import "C"

type CannyFilter struct {
	caps *mcu.GstCaps

	pipe                   *mcu.GstElement
	appSrc                 *mcu.GstElement
	queueRtpJitterBuffer   *mcu.GstElement
	rtpJitterBuffer        *mcu.GstElement
	queueRtpVP8depay       *mcu.GstElement
	rtpVP8depay            *mcu.GstElement
	queueVP8dec            *mcu.GstElement
	vp8dec                 *mcu.GstElement
	queueVideoconvertIn    *mcu.GstElement
	videoconvertIn         *mcu.GstElement
	queueVisionCannyFilter *mcu.GstElement
	visionCannyFilter      *mcu.GstElement
	queueVideoconvertOut   *mcu.GstElement
	videoconvertOut        *mcu.GstElement
	queueVP8enc            *mcu.GstElement
	vp8enc                 *mcu.GstElement
	queueAppSink           *mcu.GstElement
	appSink                *mcu.GstElement

	track *sfu.TrackContext
}

func (c *CannyFilter) Close() error {
	_ = mcu.ElementSetState(c.pipe, mcu.StateNull)

	mcu.ElementDeinit(c.pipe)
	mcu.ElementDeinit(c.appSrc)
	mcu.ElementDeinit(c.queueRtpJitterBuffer)
	mcu.ElementDeinit(c.rtpJitterBuffer)
	mcu.ElementDeinit(c.queueRtpVP8depay)
	mcu.ElementDeinit(c.rtpVP8depay)
	mcu.ElementDeinit(c.queueVP8dec)
	mcu.ElementDeinit(c.vp8dec)
	mcu.ElementDeinit(c.queueVideoconvertIn)
	mcu.ElementDeinit(c.videoconvertIn)
	mcu.ElementDeinit(c.queueVisionCannyFilter)
	mcu.ElementDeinit(c.visionCannyFilter)
	mcu.ElementDeinit(c.queueVideoconvertOut)
	mcu.ElementDeinit(c.videoconvertOut)
	mcu.ElementDeinit(c.queueVP8enc)
	mcu.ElementDeinit(c.vp8enc)
	mcu.ElementDeinit(c.queueAppSink)
	mcu.ElementDeinit(c.appSink)
	return nil
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
	state := mcu.ElementSetState(c.pipe, mcu.StatePlaying)
	if state != mcu.StateChangeReturnAsync {
		return errors.New("could not state canny filter pipeline")
	}
	go c.handleSample()
	return nil
}

func writeSampleIfExist(t *sfu.TrackContext, sink *mcu.GstElement) {
	gstSample, err := mcu.AppSinkPullSample(sink)
	if err != nil {
		log.Println(err)
		if mcu.AppSinkIsEOS(sink) == true {
			log.Println("EOS")
		} else {
			log.Println("could not get sample from sink")
		}
	}
	defer gstSample.Deinit()

	w, err := t.GetTrackRemoteWriterSample()
	if err != nil {
		log.Println("writer empty", err)
		return
	}

	err = w.WriteRemote(media.Sample{
		Data:     C.GoBytes(gstSample.Buff, C.int(gstSample.Size)),
		Duration: time.Millisecond,
	})
	if err != nil {
		log.Println("write track remote", err)
	}
}

func (c *CannyFilter) handleSample() {
	for {
		select {
		case <-c.track.Done():
			return
		default:
			writeSampleIfExist(c.track, c.appSink)
		}
	}
}

const _QUEUE_BUFFER_SIZE = 10485760 * 8

func setQueueBufferSize(elem *mcu.GstElement) {
	mcu.ObjectSet(elem, "max-size-bytes", _QUEUE_BUFFER_SIZE)
}

// TODO: use just function
func NewCannyFilter(t *sfu.TrackContext) (sfu.Pipeline, error) {
	var filter CannyFilter
	filter.track = t
	filter.caps = mcu.CapsFromString(mcu.NewRtpVP8Caps(t.GetClockRate()))

	pipe, err := mcu.PipelineNew("")
	if err != nil {
		return nil, err
	}
	filter.pipe = pipe

	src, err := mcu.ElementFactoryMake("appsrc", "")
	if err != nil {
		return nil, err
	}
	filter.appSrc = src

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

	queueVP8dec, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueVP8dec = queueVP8dec

	vp8dec, err := mcu.ElementFactoryMake("vp8dec", "")
	if err != nil {
		return nil, err
	}
	filter.vp8dec = vp8dec

	queueVideoconvertIn, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueVideoconvertIn = queueVideoconvertIn

	videoconvertIn, err := mcu.ElementFactoryMake("videoconvert", "")
	if err != nil {
		return nil, err
	}
	filter.videoconvertIn = videoconvertIn

	queueVisionCannyFilter, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueVisionCannyFilter = queueVisionCannyFilter

	visionCannyFilter, err := mcu.ElementFactoryMake("visioncannyfilter", "")
	if err != nil {
		return nil, err
	}
	filter.visionCannyFilter = visionCannyFilter

	queueVideoconvertOut, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueVideoconvertOut = queueVideoconvertOut

	videoconvertOut, err := mcu.ElementFactoryMake("videoconvert", "")
	if err != nil {
		return nil, err
	}
	filter.videoconvertOut = videoconvertOut

	queueVP8enc, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueVP8enc = queueVP8enc

	vp8enc, err := mcu.ElementFactoryMake("vp8enc", "")
	if err != nil {
		return nil, err
	}
	filter.vp8enc = vp8enc

	queueAppSink, err := mcu.ElementFactoryMake("queue", "")
	if err != nil {
		return nil, err
	}
	filter.queueAppSink = queueAppSink

	sink, err := mcu.ElementFactoryMake("appsink", "")
	if err != nil {
		return nil, err
	}
	filter.appSink = sink

	mcu.ObjectSet(src, "caps", filter.caps)
	mcu.ObjectSet(src, "format", 3)
	mcu.ObjectSet(src, "is-live", true)
	mcu.ObjectSet(src, "do-timestamp", true)

	mcu.ObjectSet(rtpJitterBuffer, "mode", 0)

	mcu.ObjectSet(vp8enc, "min-quantizer", 2)
	mcu.ObjectSet(vp8enc, "max-quantizer", 56)
	mcu.ObjectSet(vp8enc, "keyframe-max-dist", 10)
	mcu.ObjectSet(vp8enc, "threads", 16)
	mcu.ObjectSet(vp8enc, "undershoot", 100)
	mcu.ObjectSet(vp8enc, "overshoot", 10)
	mcu.ObjectSet(vp8enc, "buffer-size", 1000)
	mcu.ObjectSet(vp8enc, "buffer-initial-size", 5000)
	mcu.ObjectSet(vp8enc, "buffer-optimal-size", 600)
	mcu.ObjectSet(vp8enc, "error-resilient", 1)
	mcu.ObjectSet(vp8enc, "deadline", 1)

	mcu.ObjectSet(sink, "sync", false)
	mcu.ObjectSet(sink, "drop", true)

	setQueueBufferSize(queueRtpJitterBuffer)
	setQueueBufferSize(queueRtpVP8depay)
	setQueueBufferSize(queueVP8dec)
	setQueueBufferSize(queueVideoconvertIn)
	setQueueBufferSize(queueVisionCannyFilter)
	setQueueBufferSize(queueVideoconvertOut)
	setQueueBufferSize(queueVP8enc)
	setQueueBufferSize(queueAppSink)

	mcu.BinAddMany(
		pipe,
		src,
		queueRtpJitterBuffer,
		rtpJitterBuffer,
		queueRtpVP8depay,
		rtpVP8depay,
		queueVP8dec,
		vp8dec,
		queueVideoconvertIn,
		videoconvertIn,
		queueVisionCannyFilter,
		visionCannyFilter,
		queueVideoconvertOut,
		videoconvertOut,
		queueVP8enc,
		vp8enc,
		queueAppSink,
		sink,
	)
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
	succes = mcu.ElementLink(rtpVP8depay, queueVP8dec)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(queueVP8dec, vp8dec)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(vp8dec, queueVideoconvertIn)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(queueVideoconvertIn, videoconvertIn)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(videoconvertIn, queueVisionCannyFilter)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(queueVisionCannyFilter, visionCannyFilter)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(visionCannyFilter, queueVideoconvertOut)
	if !succes {
		panic("unable link GstElement")
	}

	// succes = mcu.ElementLink(queueVisionCannyFilter, videoconvertOut)
	// if !succes {
	// 	panic("unable link GstElement")
	// }
	succes = mcu.ElementLink(queueVideoconvertOut, videoconvertOut)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(videoconvertOut, queueVP8enc)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(queueVP8enc, vp8enc)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(vp8enc, queueAppSink)
	if !succes {
		panic("unable link GstElement")
	}
	succes = mcu.ElementLink(queueAppSink, sink)
	if !succes {
		panic("unable link GstElement")
	}

	return &filter, nil
}
