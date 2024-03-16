#ifndef PIPELINES_H
#define PIPELINES_H

#include <gst/gst.h>

#ifdef __cplusplus
extern "C" {

// Example of Caps: video/x-vp8, profile=(string)0, streamheader=(buffer)< >,
// width = (int)494, height=(int)371, pixel-aspect-ratio=(fraction)1/1,
// framerate=(fraction)0/1, interlace-mode=(string)progressive,
// colorimetry=(string)bt601, chroma-site=(string)jpeg,
// multiview-mode=(string)mono,
// multiview-flags=(GstVideoMultiviewFlagsSet)0:ffffffff:/right-view-first/left-flipped/left-flopped/right-flipped/right-flopped/half-aspect/mixed-mono
class CapsAttr {
public:
  CapsAttr(GstCaps *caps);
  virtual ~CapsAttr();

  int height();
  int width();

  GstCaps *caps;
  GstStructure *structure;
};

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

#endif

void write_pipe(void *pipe, void *buffer, int len);
void start_pipe(void *pipe);
void delete_pipe(void *pipe);

void setup();
void print_version();

#ifdef __cplusplus
}
#endif

#endif
