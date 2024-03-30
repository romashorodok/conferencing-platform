#include <glib.h>
#include <gst/app/gstappsink.h>
#include <gst/app/gstappsrc.h>
#include <gst/gst.h>
#include <iostream>

#include <dummy/dummyelements.h>
#include <dummy/dummytransform.h>

#include "pipelines.h"
#include "vision/visioncannyfilter.h"

CapsAttr::CapsAttr(GstCaps *caps) {
  this->caps = caps;
  if (this->caps == nullptr) {
    throw std::runtime_error("empty sample caps");
  }

  structure = gst_caps_get_structure(caps, 0);
  if (this->structure == nullptr) {
    throw std::runtime_error("empty sample caps");
  }
}

CapsAttr::~CapsAttr() {
  // NOTE: sample reuse alloc buffer if unref caps it will be removed
  // g_free(structure);
  // gst_caps_unref(caps);
}

int CapsAttr::width() {
  if (caps == nullptr || structure == nullptr) {
    return -1;
  }

  gint width = 0;
  if (gst_structure_get_int(structure, "width", &width)) {
    return width;
  }

  return -1;
}

int CapsAttr::height() {
  if (caps == nullptr || structure == nullptr) {
    return -1;
  }

  gint width = 0;
  if (gst_structure_get_int(structure, "height", &width)) {
    return width;
  }

  return -1;
}

BasePipeline::BasePipeline(char *trackID) {
  this->trackID = trackID;
  this->pipeline = gst_pipeline_new(nullptr);
  this->appsrc = gst_element_factory_make("appsrc", nullptr);
}

BasePipeline::~BasePipeline() {
  std::cout << "destroy from base pipe" << std::endl;
  gst_element_deinit(this->pipeline);
  gst_element_deinit(this->appsrc);
}

extern "C" void start_pipe(void *pipe) {
  auto _pipe = dynamic_cast<BasePipeline *>(static_cast<BasePipeline *>(pipe));
  if (_pipe == nullptr)
    return;

  auto pipeline = _pipe->getPipeline();

  gst_element_set_state(pipeline, GST_STATE_PLAYING);
}

extern "C" void write_pipe(void *pipe, void *buffer, int len) {
  auto _pipe = dynamic_cast<BasePipeline *>(static_cast<BasePipeline *>(pipe));
  if (_pipe == nullptr)
    return;

  auto *src = _pipe->getAppsrc();

  if (src != nullptr) {
    gpointer p = g_memdup2(buffer, len);
    GstBuffer *buffer = gst_buffer_new_wrapped(p, len);
    buffer->duration = GST_MSECOND;
    gst_app_src_push_buffer(GST_APP_SRC(src), buffer);
  }

  free(buffer);
}

#define PACKAGE "pipelines"
#define VERSION "0.0.0"

static gboolean gst_register_elements(GstPlugin *plugin) {
  gboolean result = FALSE;

  // result |= gst_element_register(plugin, "dummytransform", GST_RANK_NONE,
  //                                GST_TYPE_DUMMY_TRANSFORM);
  result |= gst_element_register(plugin, "visioncannyfilter", GST_RANK_NONE,
                                 VISION_TYPE_CANNY_FILTER);

  if (!result)
    g_assert_not_reached();

  return result;
}

#define PIPELINE_PACKAGE_NAME "conferencingpipelines"

void setup() {
  gst_init(0, nullptr);

  gst_plugin_register_static(GST_VERSION_MAJOR, GST_VERSION_MINOR,
                             "pipelinesstaticelements",
                             "pipelines static elements", gst_register_elements,
                             VERSION, GST_LICENSE_UNKNOWN, PACKAGE,
                             PIPELINE_PACKAGE_NAME, PIPELINE_PACKAGE_NAME);
}

void print_version() {
  std::cout << "hello test 1.1.10" << std::endl;

  guint major, minor, micro, nano;
  gst_version(&major, &minor, &micro, &nano);

  std::cout << "GStreamer version:"
            << " " << major << " " << minor << micro << nano << std::endl;

  GstElement *result = gst_element_factory_make("vp8enc", nullptr);

  std::cout << result << std::endl;
  std::cout << GST_ELEMENT_NAME(result) << std::endl;
}
