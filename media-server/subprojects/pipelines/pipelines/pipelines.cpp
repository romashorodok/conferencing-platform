#include "pipelines.h"
#include <glib.h>
#include <gst/gst.h>
#include <iostream>

void setup() { gst_init(0, nullptr); }

void print_version() {
  guint major, minor, micro, nano;
  gst_version(&major, &minor, &micro, &nano);

  std::cout << "GStreamer version:"
            << " " << major << " " << minor << micro << nano << std::endl;

  GstElement *result = gst_element_factory_make("vp8enc", nullptr);

  std::cout << result << std::endl;

  std::cout << GST_ELEMENT_NAME(result) << std::endl;
}

