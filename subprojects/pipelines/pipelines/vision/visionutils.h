
#include <glib.h>
#include <gst/video/video-info.h>
#include <opencv2/core.hpp>

G_BEGIN_DECLS

gboolean params_from_caps(GstCaps *caps, gint *width, gint *height,
                          int *cv_type, GError **err);

gboolean params_from_info(GstVideoInfo *info, gint *width, gint *height,
                          int *cv_type, GError **err);

gboolean cv_type_from_video_format(GstVideoFormat format, int *cv_type,
                                GError **err);

GstCaps *caps_from_cv_type(int cv_type);

G_END_DECLS
