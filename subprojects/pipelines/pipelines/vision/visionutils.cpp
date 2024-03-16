#include "visionutils.h"

gboolean params_from_caps(GstCaps *caps, gint *width, gint *height,
                          int *cv_type, GError **err) {
  GstVideoInfo info;
  gchar *caps_str;

  if (!gst_video_info_from_caps(&info, caps)) {
    caps_str = gst_caps_to_string(caps);
    GST_ERROR("Failed to get video info from caps %s", caps_str);
    g_set_error(err, GST_CORE_ERROR, GST_CORE_ERROR_NEGOTIATION,
                "Failed to get video info from caps %s", caps_str);
    g_free(caps_str);
    return FALSE;
  }

  return params_from_info(&info, width, height, cv_type, err);
}

gboolean params_from_info(GstVideoInfo *info, gint *width, gint *height,
                          int *cv_type, GError **err) {
  auto format = GST_VIDEO_INFO_FORMAT(info);

  if (!cv_type_from_video_format(format, cv_type, err)) {
    return FALSE;
  }

  *width = GST_VIDEO_INFO_WIDTH(info);
  *height = GST_VIDEO_INFO_HEIGHT(info);

  return TRUE;
}

gboolean cv_type_from_video_format(GstVideoFormat format, int *cv_type,
                                   GError **err) {
  const gchar *format_str;

  switch (format) {
  case GST_VIDEO_FORMAT_GRAY8:
    *cv_type = CV_8UC1;
    break;
  case GST_VIDEO_FORMAT_RGB:
  case GST_VIDEO_FORMAT_BGR:
    *cv_type = CV_8UC3;
    break;
  case GST_VIDEO_FORMAT_RGBx:
  case GST_VIDEO_FORMAT_xRGB:
  case GST_VIDEO_FORMAT_BGRx:
  case GST_VIDEO_FORMAT_xBGR:
  case GST_VIDEO_FORMAT_RGBA:
  case GST_VIDEO_FORMAT_ARGB:
  case GST_VIDEO_FORMAT_BGRA:
  case GST_VIDEO_FORMAT_ABGR:
    *cv_type = CV_8UC4;
    break;
  case GST_VIDEO_FORMAT_GRAY16_LE:
  case GST_VIDEO_FORMAT_GRAY16_BE:
    *cv_type = CV_16UC1;
    break;
  default:
    format_str = gst_video_format_to_string(format);
    g_set_error(err, GST_CORE_ERROR, GST_CORE_ERROR_NEGOTIATION,
                "Unsupported video format %s", format_str);
    return FALSE;
  }

  return TRUE;
}

GstCaps *type_from_int(int cv_type) {
  GstCaps *c = gst_caps_new_empty();
  switch (cv_type) {
  case CV_8UC1:
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("GRAY8")));
    break;
  case CV_8UC3:
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("RGB")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("BGR")));
    break;
  case CV_8UC4:
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("RGBx")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("xRGB")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("BGRx")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("xBGR")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("RGBA")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("ARGB")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("BGRA")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("ABGR")));
    break;
  case CV_16UC1:
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("GRAY16_LE")));
    gst_caps_append(c, gst_caps_from_string(GST_VIDEO_CAPS_MAKE("GRAY16_BE")));
    break;
  }
  return c;
}
