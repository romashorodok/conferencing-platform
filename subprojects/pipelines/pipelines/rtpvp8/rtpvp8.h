#include <gst/gst.h>
#include <pipelines/pipelines.h>

#ifdef __cplusplus
extern "C" {

class RtpVP8 : public BasePipeline {
public:
  RtpVP8(char *trackID);
  ~RtpVP8();

  inline GstElement *getCgoOnSampleSink() const { return cgoOnSampleSink; }

private:
  GstElement *rtpsession;
  GstElement *queueRtpvp8depay;
  GstElement *rtpvp8depay;
  GstElement *queueVp8dec;
  GstElement *vp8dec;
  GstElement *videoconvertIn;
  GstElement *videoscale;
  GstElement *queueDummyTransform;
  GstElement *dummyTransform;
  GstElement *videoconvertOut;
  GstElement *queueVp8enc;
  GstElement *vp8enc;
  GstElement *cgoOnSampleSink;
};

#endif
extern void CGO_rtp_vp8_dummy_sample(char *trackID, void *buffer, int size,
                                     int duration);

void *new_pipe_rtp_vp8(char *track);
#ifdef __cplusplus
}
#endif
