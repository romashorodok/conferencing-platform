#include "visionframe.h"
#include "visionutils.h"
#include <iostream>

G_DEFINE_ABSTRACT_TYPE(FRAME_TYPE_NAME, FRAME_NAME, GST_TYPE_VIDEO_FILTER);

static gboolean vision_frame_set_info(GstVideoFilter *trans, GstCaps *incaps,
                                      GstVideoInfo *in_info, GstCaps *outcaps,
                                      GstVideoInfo *out_info) {
  std::cout << "vision frqme set called" << std::endl;

  auto frame = VISION_FRAME(trans);
  auto frameClass = VISION_FRAME_GET_CLASS(frame);

  gint in_width, in_height;
  int in_type;
  GError *in_err = nullptr;

  if (!params_from_info(in_info, &in_width, &in_height, &in_type, &in_err)) {
    GST_WARNING_OBJECT(trans, "Failed to parse input info: %s",
                       in_err->message);
    g_error_free(in_err);
    return FALSE;
  }

  gint out_width, out_height;
  int out_type;
  GError *out_err = nullptr;

  if (!params_from_info(out_info, &out_width, &out_height, &out_type,
                        &out_err)) {
    GST_WARNING_OBJECT(trans, "Failed to parse output info: %s",
                       out_err->message);
    g_error_free(out_err);
    return FALSE;
  }

  if (frameClass->on_caps) {
    if (!frameClass->on_caps(frame, in_width, in_height, in_type, out_width,
                             out_height, out_type)) {
      return FALSE;
    }
  }

  frame->inframe.create(cv::Size(in_width, in_height), in_type);
  frame->outframe.create(cv::Size(out_width, out_height), out_type);

  return TRUE;
}

static GstFlowReturn vision_frame_transform_frame(GstVideoFilter *trans,
                                                  GstVideoFrame *inframe,
                                                  GstVideoFrame *outframe) {

  auto frame = VISION_FRAME(trans);
  auto frameClass = VISION_FRAME_GET_CLASS(frame);

  if (frame == nullptr) {
    g_error("frame is null ptr");
    g_assert_not_reached();
  }

  if (frameClass == nullptr) {
    g_error("frame class is null ptr");
    g_assert_not_reached();
  }

  if (frameClass->on_frame == nullptr) {
    g_error("frame class on_frame is null ptr");
    g_assert_not_reached();
  }

  frame->inframe.data = static_cast<unsigned char *>(inframe->data[0]);
  frame->inframe.datastart = static_cast<unsigned char *>(inframe->data[0]);
  frame->outframe.data = static_cast<unsigned char *>(outframe->data[0]);
  frame->outframe.datastart = static_cast<unsigned char *>(outframe->data[0]);

  return frameClass->on_frame(frame, inframe->buffer, frame->inframe,
                              outframe->buffer, frame->outframe);
}

static void vision_frame_finalize(GObject *obj) {
  auto frame = VISION_FRAME(obj);
  frame->inframe.release();
  frame->outframe.release();

  G_OBJECT_CLASS(vision_frame_parent_class)->finalize(obj);
}

static void vision_frame_class_init(VisionFrameClass *klass) {
  auto video_filter = GST_VIDEO_FILTER_CLASS(klass);
  video_filter->transform_frame = vision_frame_transform_frame;
  video_filter->set_info = vision_frame_set_info;

  auto gobject_class = G_OBJECT_CLASS(klass);
  gobject_class->finalize = GST_DEBUG_FUNCPTR(vision_frame_finalize);
}

static void vision_frame_init(VisionFrame *self) {}
