#include <gst/app/gstappsink.h>
#include <gst/app/gstappsrc.h>
#include <gst/gst.h>
#include <iostream>

#include <opencv2/imgproc.hpp>
#include <opencv2/opencv.hpp>

// #include "cannyfilter.h"
#include "rtpvp8.h"

// static inline cv::Mat *gst_buffer_to_mathExp(GstBuffer *buf, GstCaps *caps) {
//   return new cv::Mat{};
// }
//
using namespace cv;

// GstFlowReturn on_raw_sample(GstAppSink *sink, gpointer pipeRaw) {
//   GstSample *sample = gst_app_sink_pull_sample(sink);
//   // auto pipe = static_cast<RtpVP8 *>(pipeRaw);
//
//   try {
//     // auto *caps = new SampleCapsAttr(sample);
//
//     if (sample) {
//       GstBuffer *buffer = gst_sample_get_buffer(sample);
//
//       if (buffer) {
//         GstMapInfo map;
//         if (gst_buffer_map(buffer, &map, GST_MAP_READ)) {
//           cv::Mat frame(cv::Size(caps->width(), caps->height()), CV_8UC3,
//                         (void *)map.data);
//
//           cv::Mat gray;
//           cvtColor(frame, gray, cv::COLOR_BGR2GRAY);
//
//           // Apply Canny edge detection
//           cv::Mat edges;
//           Canny(gray, edges, 50, 150);
//
//           std::cout << edges << std::endl;
//
//           // Release GstBuffer map
//           gst_buffer_unmap(buffer, &map);
//         }
//       }
//
//       gst_sample_unref(sample);
//     }
//
//     delete caps;
//   } catch (const std::exception &e) {
//     std::cout << e.what() << std::endl;
//   }
//
//   return GST_FLOW_OK;
// }

GstFlowReturn on_sample_vp8_pipeline(GstAppSink *sink, gpointer pipeRaw) {
  GstSample *sample = gst_app_sink_pull_sample(sink);

  auto pipe = static_cast<RtpVP8 *>(pipeRaw);

  if (sample) {
    GstBuffer *buffer = gst_sample_get_buffer(sample);

    if (buffer) {
      gpointer copy = nullptr;
      gsize copy_size = 0;

      // g_print("Received sample with size: %lu, timestamp: %llu \n",
      //         gst_buffer_get_size(buffer), GST_BUFFER_TIMESTAMP(buffer));

      gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy,
                             &copy_size);

      CGO_rtp_vp8_dummy_sample(pipe->getTrackID(), copy, copy_size,
                               buffer->duration);
    }

    gst_sample_unref(sample);
  }

  return GST_FLOW_OK;
}

RtpVP8::RtpVP8(char *trackID) : BasePipeline(trackID) {
  auto appsrc = this->getAppsrc();
  g_object_set(appsrc, "format", GST_FORMAT_TIME, "is-live", TRUE, nullptr);
  g_object_set(appsrc, "do-timestamp", TRUE, nullptr);

  GstCaps *caps = gst_caps_new_simple(
      "application/x-rtp", "media", G_TYPE_STRING, "video", "payload",
      G_TYPE_INT, 96, "clock-rate", G_TYPE_INT, 90000, "encoding-name",
      G_TYPE_STRING, "VP8-DRAFT-IETF-01", nullptr);
  g_object_set(appsrc, "caps", caps, nullptr);
  gst_caps_unref(caps);

  this->queueRtpSession = gst_element_factory_make("queue", nullptr);
  g_object_set(queueRtpSession, "max-size-bytes", guint(10485760 * 8), nullptr);

  this->rtpsession = gst_element_factory_make("rtpjitterbuffer", nullptr);
  // Add GstReferenceTimestampMeta to buffers with the original reconstructed
  // reference clock timestamp.
  // g_object_set(rtpsession, "add-reference-timestamp-meta", TRUE, nullptr);
  // g_object_set(rtpsession, "drop-on-latency", TRUE, nullptr);
  g_object_set(rtpsession, "mode", 0, nullptr);

  this->identity = gst_element_factory_make("identity", nullptr);

  this->queueRtpvp8depay = gst_element_factory_make("queue", nullptr);
  g_object_set(queueRtpvp8depay, "max-size-bytes", guint(10485760 * 8),
               nullptr);
  this->rtpvp8depay = gst_element_factory_make("rtpvp8depay", nullptr);
  g_object_set(rtpvp8depay, "auto-header-extension", TRUE, nullptr);
  this->queueVp8dec = gst_element_factory_make("queue", nullptr);
  g_object_set(queueVp8dec, "max-size-bytes", guint(10485760 * 8), nullptr);
  this->vp8dec = gst_element_factory_make("vp8dec", nullptr);
  g_object_set(vp8dec, "threads", 16, nullptr);
  this->queueVideoconvertIn = gst_element_factory_make("queue", nullptr);
  g_object_set(queueVideoconvertIn, "max-size-bytes", guint(10485760 * 8),
               nullptr);
  this->videoconvertIn = gst_element_factory_make("videoconvert", nullptr);
  this->queueDummyTransform = gst_element_factory_make("queue", nullptr);
  g_object_set(queueDummyTransform, "max-size-bytes", guint(10485760 * 8),
               nullptr);
  // this->dummyTransform = gst_element_factory_make("visioncannyfilter",
  // nullptr); this->dummyTransform = gst_element_factory_make("edgedetect",
  // nullptr); this->dummyTransform = gst_element_factory_make("dummytransform",
  // nullptr);
  this->videoconvertOut = gst_element_factory_make("videoconvert", nullptr);

  this->queueVp8enc = gst_element_factory_make("queue", nullptr);
  g_object_set(queueVp8enc, "max-size-bytes", guint(10485760 * 8), nullptr);
  this->vp8enc = gst_element_factory_make("vp8enc", nullptr);

  g_object_set(vp8enc, "min-quantizer", 2, nullptr);
  g_object_set(vp8enc, "max-quantizer", 56, nullptr);
  // g_object_set(vp8enc, "static-threshold", 1, nullptr);
  g_object_set(vp8enc, "keyframe-max-dist", 10, nullptr);
  g_object_set(vp8enc, "threads", 16, nullptr);

  g_object_set(vp8enc, "undershoot", 100, nullptr);
  g_object_set(vp8enc, "overshoot", 10, nullptr);

  g_object_set(vp8enc, "buffer-size", 1000, nullptr);
  g_object_set(vp8enc, "buffer-initial-size", 5000, nullptr);
  g_object_set(vp8enc, "buffer-optimal-size", 600, nullptr);

  // Pictures are composed of slices that can be highly flexible in shape, and
  // each slice of a picture is coded completely independently of the other
  // slices in the same picture to enable enhanced loss/error resilience.
  g_object_set(vp8enc, "error-resilient", 1, nullptr);
  // // Maximum distance between keyframes (number of frames)
  // g_object_set(vp8enc, "keyframe-max-dist", 10, nullptr);
  // // Automatically generate AltRef frames
  // // When --auto-alt-ref is enabled the default mode of operation is to
  // either
  // // populate the buffer with a copy of the previous golden frame when this
  // // frame is updated, or with a copy of a frame derived from some point of
  // // time in the future (the choice is made automatically by the encoder).
  // g_object_set(vp8enc, "auto-alt-ref", TRUE, nullptr);
  g_object_set(vp8enc, "deadline", 1, nullptr);

  // customprocessor
  this->cgoOnSampleSink = gst_element_factory_make("appsink", nullptr);
  g_object_set(cgoOnSampleSink, "sync", FALSE, nullptr);
  g_object_set(cgoOnSampleSink, "drop", TRUE, nullptr);

  if (!this->queueRtpSession || !this->rtpsession || !this->identity ||
      !this->queueRtpvp8depay || !this->rtpvp8depay || !this->queueVp8dec ||
      !this->vp8dec || !this->queueVideoconvertIn || !this->vp8enc ||
      !this->queueDummyTransform ||
      // !this->dummyTransform ||
      !this->videoconvertIn || !this->videoconvertOut || !this->queueVp8enc ||
      !this->cgoOnSampleSink) {
    g_printerr("Unable create the pipeline.\n");
    exit(1);
  }

  gst_bin_add_many(
      GST_BIN(this->getPipeline()), this->getAppsrc(), this->queueRtpSession,
      this->rtpsession, this->identity, this->queueRtpvp8depay,
      this->rtpvp8depay, this->queueVp8dec, this->vp8dec,
      this->queueVideoconvertIn, this->videoconvertIn,
      this->queueDummyTransform,
      // this->dummyTransform,
    this->videoconvertOut, this->queueVp8enc,
      this->vp8enc, this->cgoOnSampleSink, nullptr);

  gst_element_link_many(
      this->getAppsrc(), this->queueRtpSession, this->rtpsession,
      this->identity, this->queueRtpvp8depay, this->rtpvp8depay,
      this->queueVp8dec, this->vp8dec, this->queueVideoconvertIn,
      this->videoconvertIn, this->queueDummyTransform,
    // this->dummyTransform,
      this->videoconvertOut, this->queueVp8enc, this->vp8enc,
      this->cgoOnSampleSink, nullptr);

  // g_object_set(cgoOnSampleSink, "emit-signals", TRUE, NULL);
  // g_signal_connect(cgoOnSampleSink, "new-sample",
  //                  G_CALLBACK(on_sample_vp8_pipeline), nullptr);

  GstAppSinkCallbacks callbacks = {NULL, NULL, on_sample_vp8_pipeline, NULL};
  // GstAppSinkCallbacks callbacks = {NULL, NULL, on_sample_vp8_pipeline, NULL};
  gst_app_sink_set_callbacks(GST_APP_SINK(this->cgoOnSampleSink), &callbacks,
                             this, NULL);
}

RtpVP8::~RtpVP8() {
  std::cout << "destory from rtp VP8 pipe" << std::endl;
  gst_element_deinit(queueRtpSession);
  gst_element_deinit(rtpsession);
  gst_element_deinit(identity);
  gst_element_deinit(queueRtpvp8depay);
  gst_element_deinit(rtpvp8depay);
  gst_element_deinit(queueVp8dec);
  gst_element_deinit(vp8dec);
  gst_element_deinit(queueVideoconvertIn);
  gst_element_deinit(videoconvertIn);
  gst_element_deinit(queueDummyTransform);
  gst_element_deinit(dummyTransform);
  gst_element_deinit(videoconvertOut);
  gst_element_deinit(queueVp8enc);
  gst_element_deinit(vp8enc);
  gst_element_deinit(cgoOnSampleSink);
}

extern "C" void delete_pipe(void *pipe) {
  auto _pipe = dynamic_cast<BasePipeline *>(static_cast<BasePipeline *>(pipe));
  if (_pipe == nullptr)
    return;
  delete _pipe;
}

extern "C" void *new_pipe_rtp_vp8(char *trackID) { return new RtpVP8(trackID); }
