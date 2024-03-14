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
  BasePipeline(char *trackID);
  virtual ~BasePipeline();

  inline GstElement *getPipeline() const { return pipeline; }
  inline GstElement *getAppsrc() const { return appsrc; }
  inline char *getTrackID() const { return trackID; }

private:
  char *trackID;

  GstElement *pipeline;
  GstElement *appsrc;
};

class RtpVP8 : public BasePipeline {
public:
  RtpVP8(char *trackID);
  ~RtpVP8();

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
extern void CGO_rtp_vp8_dummy_sample(char *trackID, void *buffer, int size,
                                     int duration);

void write_pipe(void *pipe, void *buffer, int len);
void start_pipe(void *pipe);
void delete_pipe(void *pipe);
void *new_pipe_rtp_vp8(char *track);
#ifdef __cplusplus
}
#endif
#endif
