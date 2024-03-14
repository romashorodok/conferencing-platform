#include <gst/app/gstappsink.h>
#include <gst/app/gstappsrc.h>
#include <gst/gst.h>
#include <iostream>

#include "rtpvp8.h"

BasePipeline::BasePipeline(char *trackID) {
  this->trackID = trackID;
  this->pipeline = gst_pipeline_new(nullptr);
  this->appsrc = gst_element_factory_make("appsrc", nullptr);
}

BasePipeline::~BasePipeline() {
  std::cout << "destroy from base pipe" << std::endl;
  gst_element_deinit(this->pipeline);
  gst_element_deinit(this->appsrc);
}

GstFlowReturn on_sample_vp8_pipeline(GstAppSink *sink, gpointer pipeRaw) {
  GstSample *sample = gst_app_sink_pull_sample(sink);
  auto pipe = static_cast<RtpVP8 *>(pipeRaw);

  // std::cout << pipe->getTrackID() << std::endl;
  // std::cout << pipe->getTrackPool() << std::endl;

  // std::cout << trackID << std::endl;
  // std::cout << "test1  test" << std::endl;
  // std::cout << "--test1" << std::endl;
  if (sample) {
    // Get sample details
    // GstCaps *caps = gst_sample_get_caps(sample);
    GstBuffer *buffer = gst_sample_get_buffer(sample);

    // Print sample details
    // if (caps) {
    //   gchar *caps_str = gst_caps_to_string(caps);
    // g_print("Received sample with caps: %s\n", caps_str);
    // g_free(caps_str);
    // }

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

  this->rtpsession = gst_element_factory_make("rtpjitterbuffer", nullptr);
  // Add GstReferenceTimestampMeta to buffers with the original reconstructed
  // reference clock timestamp.
  g_object_set(rtpsession, "add-reference-timestamp-meta", TRUE, nullptr);
  g_object_set(rtpsession, "drop-on-latency", TRUE, nullptr);
  g_object_set(rtpsession, "mode", 0, nullptr);

  this->queueRtpvp8depay = gst_element_factory_make("queue", nullptr);
  this->rtpvp8depay = gst_element_factory_make("rtpvp8depay", nullptr);
  g_object_set(rtpvp8depay, "auto-header-extension", TRUE, nullptr);
  this->queueVp8dec = gst_element_factory_make("queue", nullptr);
  this->vp8dec = gst_element_factory_make("vp8dec", nullptr);
  this->vp8enc = gst_element_factory_make("vp8enc", nullptr);

  // Pictures are composed of slices that can be highly flexible in shape, and
  // each slice of a picture is coded completely independently of the other
  // slices in the same picture to enable enhanced loss/error resilience.
  g_object_set(vp8enc, "error-resilient", 1, nullptr);
  // Maximum distance between keyframes (number of frames)
  g_object_set(vp8enc, "keyframe-max-dist", 10, nullptr);
  // Automatically generate AltRef frames
  // When --auto-alt-ref is enabled the default mode of operation is to either
  // populate the buffer with a copy of the previous golden frame when this
  // frame is updated, or with a copy of a frame derived from some point of time
  // in the future (the choice is made automatically by the encoder).
  g_object_set(vp8enc, "auto-alt-ref", TRUE, nullptr);
  g_object_set(vp8enc, "deadline", 1, nullptr);

  // this->webmmux = gst_element_factory_make("webmmux", nullptr);
  // g_object_set(this->webmmux, "streamable", TRUE, nullptr);
  // this->filesink = gst_element_factory_make("filesink", nullptr);
  // g_object_set(this->filesink, "location", "proxy.webm", nullptr);

  this->cgoOnSampleSink = gst_element_factory_make("appsink", nullptr);

  if (!this->rtpsession || !this->queueRtpvp8depay || !this->rtpvp8depay ||
      !this->queueVp8dec || !this->vp8dec || !this->vp8enc ||
      !this->cgoOnSampleSink
      // !this->webmmux ||
      // !this->filesink
  ) {
    g_printerr("Unable create the pipeline.\n");
    exit(1);
  }

  gst_bin_add_many(GST_BIN(this->getPipeline()), this->getAppsrc(),
                   this->rtpsession, this->queueRtpvp8depay, this->rtpvp8depay,
                   this->queueVp8dec, this->vp8dec, this->vp8enc,
                   // this->webmmux, this->filesink,

                   this->cgoOnSampleSink, nullptr);

  gst_element_link_many(this->getAppsrc(), this->rtpsession,
                        this->queueRtpvp8depay, this->rtpvp8depay,
                        this->queueVp8dec, this->vp8dec, this->vp8enc,
                        // this->webmmux, this->filesink,

                        this->cgoOnSampleSink, nullptr);

  // g_object_set(cgoOnSampleSink, "emit-signals", TRUE, NULL);
  // g_signal_connect(cgoOnSampleSink, "new-sample",
  //                  G_CALLBACK(gstreamer_send_new_sample_handler), nullptr);

  GstAppSinkCallbacks callbacks = {NULL, NULL, on_sample_vp8_pipeline, NULL};
  gst_app_sink_set_callbacks(GST_APP_SINK(this->cgoOnSampleSink), &callbacks,
                             this, NULL);
}

RtpVP8::~RtpVP8() {
  std::cout << "destory from rtp VP8 pipe" << std::endl;
  // gst_element_deinit(this->rtpsession);
  gst_element_deinit(this->queueRtpvp8depay);
  gst_element_deinit(this->rtpvp8depay);
  gst_element_deinit(this->queueVp8dec);
  gst_element_deinit(this->vp8dec);
  // gst_element_deinit(this->printsink);
}

extern "C" void start_pipe(void *pipe) {
  auto _pipe = dynamic_cast<BasePipeline *>(static_cast<BasePipeline *>(pipe));
  if (_pipe == nullptr)
    return;

  auto pipeline = _pipe->getPipeline();

  gst_element_set_state(pipeline, GST_STATE_PLAYING);
}

extern "C" void write_pipe(void *pipe, void *buffer, int len) {
  auto _pipe = dynamic_cast<BasePipeline *>(static_cast<BasePipeline *>(pipe));
  if (_pipe == nullptr)
    return;

  auto *src = _pipe->getAppsrc();

  // std::cout << "write_proxy_pipe" << std::endl;

  if (src != nullptr) {
    // Assuming buffer is the data you want to push

    // // std::cout << "gst_app_src_push_buffer" << std::endl;
    gpointer p = g_memdup2(buffer, len);
    GstBuffer *buffer = gst_buffer_new_wrapped(p, len);
    // buffer->duration = GST_MSECOND;
    gst_app_src_push_buffer(GST_APP_SRC(src), buffer);
  }

  free(buffer);
}

extern "C" void delete_pipe(void *pipe) {
  auto _pipe = dynamic_cast<BasePipeline *>(static_cast<BasePipeline *>(pipe));
  if (_pipe == nullptr)
    return;
  delete _pipe;
}

extern "C" void *new_pipe_rtp_vp8(char *trackID) { return new RtpVP8(trackID); }
