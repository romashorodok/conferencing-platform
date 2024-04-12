#ifndef __VISION_FRAME_H__
#define __VISION_FRAME_H__

#include <glib.h>
#include <gst/gst.h>
#include <gst/video/gstvideofilter.h>
#include <opencv2/imgproc.hpp>

G_BEGIN_DECLS

#define FRAME_NAME vision_frame
#define FRAME_TYPE_NAME VisionFrame
#define VISION_TYPE_FRAME (vision_frame_get_type())

typedef struct _VisionFrame VisionFrame;
typedef struct _VisionFrameClass VisionFrameClass;
typedef struct _VisionFramePrivate VisionFramePrivate;

typedef GstFlowReturn (*VisionFrameTransformFn)(VisionFrame *frame,
                                                GstBuffer *buffer, cv::Mat img,
                                                GstBuffer *outbuf,
                                                cv::Mat outimg);

typedef gboolean (*VisionFrameCapsFn)(VisionFrame *frame, gint in_width,
                                      gint in_height, int in_cv_type,
                                      gint out_width, gint out_height,
                                      int out_cv_type);

#define VISION_FRAME(obj)                                                      \
  (G_TYPE_CHECK_INSTANCE_CAST((obj), VISION_TYPE_FRAME, VisionFrame))
#define VISION_FRAME_CLASS(klass)                                              \
  (G_TYPE_CHECK_CLASS_CAST((klass), VISION_TYPE_FRAME, VisionFrameClass))
#define VISION_IS_FRAME(obj)                                                   \
  (G_TYPE_CHECK_INSTANCE_TYPE((obj), VISION_TYPE_FRAME))
#define VISION_IS_FRAME_CLASS(klass)                                           \
  (G_TYPE_CHECK_CLASS_TYPE((klass), VISION_TYPE_FRAME))
#define VISION_FRAME_CAST(obj) ((VisionFrame *)(obj))
#define VISION_FRAME_GET_CLASS(obj)                                            \
  (G_TYPE_INSTANCE_GET_CLASS((obj), VISION_TYPE_FRAME, VisionFrameClass))

GType vision_frame_get_type(void);

struct _VisionFrame {
  GstVideoFilter trans;

  cv::Mat inframe;
  cv::Mat outframe;
};

struct _VisionFrameClass {
  GstVideoFilterClass parent_class;

  VisionFrameCapsFn on_caps;
  VisionFrameTransformFn on_frame;
};

G_END_DECLS

#endif
