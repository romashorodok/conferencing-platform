#ifndef PROXY_PIPE_H
#define PROXY_PIPE_H

#include <gst/gst.h>

#ifdef __cplusplus
extern "C" {

inline void gst_element_deinit(GstElement *elem) {
  if (elem == nullptr)
    return;
  gst_object_unref(elem);
}

class BasePipeline {
public:
  BasePipeline(const char *name);
  virtual ~BasePipeline();

  inline GstElement *getPipeline() const { return pipeline; }
  inline GstElement *getAppsrc() const { return appsrc; }

private:
  GstElement *pipeline;
  GstElement *appsrc;
};

class ProxyPipeRtpVP8 : public BasePipeline {
public:
  ProxyPipeRtpVP8(const char *name);
  ~ProxyPipeRtpVP8();

private:
  GstElement *testsrc;
  GstElement *rtpsession;
  GstElement *queueRtpvp8depay;
  GstElement *rtpvp8depay;
  GstElement *queueVp8dec;
  GstElement *vp8dec;
  GstElement *vp8enc;
  GstElement *webmmux;
  // GstElement *filesink;
  GstElement *cgoOnSampleSink;
};

#endif
extern void CGO_onSampleBuffer(void *buffer, int size, int duration);

void write_proxy_pipe(void *pipe, void *buffer, int len);
void start_proxy_pipe(void *pipe);
void delete_proxy_pipe(void *pipe);
void *new_proxy_pipe_rtp_vp8(const char *name);
#ifdef __cplusplus
}
#endif
#endif

