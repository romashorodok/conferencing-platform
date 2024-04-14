#ifndef __VISION_CANNY_FILTER_H__
#define __VISION_CANNY_FILTER_H__

#include "glib/glib-object.h"
#include <glib.h>
#include <gst/gst.h>
#include <gst/video/gstvideofilter.h>
#include <opencv2/imgproc.hpp>

#include "visionframe.h"

G_BEGIN_DECLS

#define CANNY_FILTER_NAME vision_canny_filter
#define CANNY_FILTER_TYPE_NAME VisionCannyFilter
#define VISION_TYPE_CANNY_FILTER (vision_canny_filter_get_type())

typedef struct _VisionCannyFilter VisionCannyFilter;
typedef struct _VisionCannyFilterClass VisionCannyFilterClass;
typedef struct _VisionCannyFilterPrivate VisionCannyFilterPrivate;

#define VISION_CANNY_FILTER(obj)                                               \
  (G_TYPE_CHECK_INSTANCE_CAST((obj), VISION_TYPE_CANNY_FILTER,                 \
                              VisionCannyFilter))
#define VISION_CANNY_FILTER_CLASS(klass)                                       \
  (G_TYPE_CHECK_CLASS_CAST((klass), VISION_TYPE_CANNY_FILTER,                  \
                           VisionCannyFilterClass))
#define VISION_IS_CANNY_FILTER(obj)                                            \
  (G_TYPE_CHECK_INSTANCE_TYPE((obj), VISION_TYPE_CANNY_FILTER))
#define VISION_IS_CANNY_FILTER_CLASS(klass)                                    \
  (G_TYPE_CHECK_CLASS_TYPE((klass), VISION_TYPE_CANNY_FILTER))
#define VISION_CANNY_FILTER_CAST(obj) ((VisionCannyFilter *)(obj))

GType vision_canny_filter_get_type();

struct _VisionCannyFilter {
  VisionFrame base;

  cv::Mat cvEdge;
  cv::Mat cvGray;
};

struct _VisionCannyFilterClass {
  VisionFrameClass parent_class;
};

G_END_DECLS

#endif
