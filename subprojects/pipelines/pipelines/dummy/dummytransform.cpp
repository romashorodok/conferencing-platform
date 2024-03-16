#include <gst/base/gstbasetransform.h>
#include <opencv2/imgproc.hpp>
#include <opencv2/opencv.hpp>

#include "../pipelines.h"
#include "dummytransform.h"

static GstStaticPadTemplate gst_dummy_transform_src_template =
    GST_STATIC_PAD_TEMPLATE("src", GST_PAD_SRC, GST_PAD_ALWAYS,
                            GST_STATIC_CAPS("video/x-raw(ANY)"));

static GstStaticPadTemplate gst_dummy_transform_sink_template =
    GST_STATIC_PAD_TEMPLATE("sink", GST_PAD_SINK, GST_PAD_ALWAYS,
                            GST_STATIC_CAPS("video/x-raw(ANY)"));

struct _GstDummyTransformPrivate {
  GstCaps *caps;
  GMutex mutex;
};

static void gst_dummy_transform_init(GstDummyTransform *);
static void gst_dummy_transform_class_init(GstDummyTransformClass *);
static GstFlowReturn gst_dummy_transform_transform_ip(GstBaseTransform *,
                                                      GstBuffer *);

G_DEFINE_TYPE(DUMMY_TRANSFORM_TYPE_NAME, DUMMY_TRANSFORM_NAME,
              GST_TYPE_BASE_TRANSFORM);

static void gst_dummy_transform_init(GstDummyTransform *self) {
  // auto *priv = self->priv = static_cast<GstDummyTransformPrivate *>(
  //     gst_dummy_transform_get_instance_private(self));
  //
  // g_mutex_init(&priv->mutex);
  //
  // g_print("Element inited", self->basetransform);
  //
}

static void gst_dummy_transform_class_init(GstDummyTransformClass *klass) {
  auto element_class = GST_ELEMENT_CLASS(klass);
  auto base_transform_class = GST_BASE_TRANSFORM_CLASS(klass);
  // auto gobject_class = G_OBJECT_CLASS(klass);

  gst_element_class_add_static_pad_template(GST_ELEMENT_CLASS(klass),
                                            &gst_dummy_transform_src_template);

  gst_element_class_add_static_pad_template(GST_ELEMENT_CLASS(klass),
                                            &gst_dummy_transform_sink_template);

  base_transform_class->transform_ip =
      GST_DEBUG_FUNCPTR(gst_dummy_transform_transform_ip);

  const gchar *name = "dummy transformer";
  gst_element_class_set_static_metadata(element_class, name, name, name, name);
}

static inline void print_caps(GstCaps *caps) {
  if (caps != nullptr) {
    gchar *capsString = gst_caps_to_string(caps);
    g_print("Caps: %s\n", capsString);
    g_free(capsString);
  } else {
    g_print("No caps available\n");
  }
}

static GstFlowReturn gst_dummy_transform_transform_ip(GstBaseTransform *trans,
                                                      GstBuffer *buf) {
  // g_return_if_fail(GST_IS_DUMMY_TRANSFORM(dummytransform));
  // g_return_if_fail(callbacks != NULL);
  auto _caps = gst_pad_get_current_caps(trans->sinkpad);
  print_caps(_caps);

  auto caps = new CapsAttr(_caps);

  std::cout << "Got buffer with: " << caps->width() << "x" << caps->height()
            << "\n"
            << std::endl;

  return GST_FLOW_OK;
}
