#include "visioncannyfilter.h"
#include <iostream>

static GstStaticPadTemplate sink_factory =
    GST_STATIC_PAD_TEMPLATE("sink", GST_PAD_SINK, GST_PAD_ALWAYS,
                            GST_STATIC_CAPS(GST_VIDEO_CAPS_MAKE("RGB")));

static GstStaticPadTemplate src_factory =
    GST_STATIC_PAD_TEMPLATE("src", GST_PAD_SRC, GST_PAD_ALWAYS,
                            GST_STATIC_CAPS(GST_VIDEO_CAPS_MAKE("RGB")));

G_DEFINE_TYPE(CANNY_FILTER_TYPE_NAME, CANNY_FILTER_NAME, VISION_TYPE_FRAME)

static GstFlowReturn vision_canny_filter_on_frame(VisionFrame *base,
                                                  GstBuffer *buf, cv::Mat img,
                                                  GstBuffer *outbuf,
                                                  cv::Mat outimg) {
  auto filter = VISION_CANNY_FILTER(base);

  cv::cvtColor(img, filter->cvGray, cv::COLOR_BGR2GRAY);

  cv::medianBlur(filter->cvGray, filter->cvGray, 3);

  cv::Canny(filter->cvGray, filter->cvEdge, 60, 160, 3);

  outimg.setTo(cv::Scalar::all(0));
  img.copyTo(outimg, filter->cvEdge);

  return GST_FLOW_OK;
}

static gboolean vision_canny_filter_on_caps(VisionFrame *transform,
                                            gint in_width, gint in_height,
                                            int in_cv_type, gint out_width,
                                            gint out_height, int out_cv_type) {
  auto filter = VISION_CANNY_FILTER(transform);

  filter->cvGray.create(cv::Size(in_width, in_height), CV_8UC1);
  filter->cvEdge.create(cv::Size(in_width, in_height), CV_8UC1);

  std::cout << in_width << " " << in_height << std::endl;

  return TRUE;
}

static void vision_canny_filter_finalize(GObject *obj) {
  auto filter = VISION_CANNY_FILTER(obj);

  std::cout << "release filter" << std::endl;

  filter->cvEdge.release();
  filter->cvGray.release();

  G_OBJECT_CLASS(vision_canny_filter_parent_class)->finalize(obj);
}

static void vision_canny_filter_class_init(VisionCannyFilterClass *klass) {

  auto frame = VISION_FRAME_CLASS(klass);

  frame->on_caps = vision_canny_filter_on_caps;
  frame->on_frame = vision_canny_filter_on_frame;

  gst_element_class_add_static_pad_template(GST_ELEMENT_CLASS(klass),
                                            &sink_factory);
  gst_element_class_add_static_pad_template(GST_ELEMENT_CLASS(klass),
                                            &src_factory);

  auto gobject_class = G_OBJECT_CLASS(klass);
  gobject_class->finalize = GST_DEBUG_FUNCPTR(vision_canny_filter_finalize);

  auto element_class = GST_ELEMENT_CLASS(klass);
  gst_element_class_set_static_metadata(element_class, "canny filter",
                                        "vision element",
                                        "perform canny filter on frame", "-");
}

static void vision_canny_filter_init(VisionCannyFilter *self) {}
