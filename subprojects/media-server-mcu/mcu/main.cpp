#include <iostream>

#include "main.h"
#include "vision/visioncannyfilter.h"

#define PACKAGE "mcu"
#define VERSION "0.0.0"
#define PACKAGE_NAME "mediaservermcu"

static gboolean gst_register_elements(GstPlugin *plugin) {
  gboolean result = FALSE;

  result |= gst_element_register(plugin, "visioncannyfilter", GST_RANK_NONE,
                                 VISION_TYPE_CANNY_FILTER);

  if (!result)
    g_assert_not_reached();

  return result;
}

extern "C" void MCU_setup() {
  gst_init(0, nullptr);

  gst_plugin_register_static(
      GST_VERSION_MAJOR, GST_VERSION_MINOR, "mediaservermcustaticelements",
      "media server mcu static elements", gst_register_elements, VERSION,
      GST_LICENSE_UNKNOWN, PACKAGE, PACKAGE_NAME, PACKAGE_NAME);
}

extern "C" void MCU_version() {
  guint major, minor, micro, nano;
  gst_version(&major, &minor, &micro, &nano);
  std::cout << "GStreamer version:"
            << " " << major << " " << minor << micro << nano << std::endl;
}

extern "C" GstBuffer *MCU_gst_buffer_new_wrapped(gchar *src, gsize len) {
  auto *dst = gst_buffer_new_allocate(NULL, len, NULL);
  gst_buffer_fill(dst, 0, src, len);
  return dst;
}

extern "C" void MCU_gst_bin_add(GstElement *p, GstElement *element) {
  gst_bin_add(GST_BIN(p), element);
}

extern "C" void MCU_gst_elem_set_string(GstElement *elem, const gchar *p_name,
                                        gchar *p_value) {
  g_object_set(G_OBJECT(elem), p_name, p_value, NULL);
}

extern "C" void MCU_gst_elem_set_int(GstElement *elem, const gchar *p_name,
                                     gint p_value) {
  g_object_set(G_OBJECT(elem), p_name, p_value, NULL);
}

extern "C" void MCU_gst_elem_set_uint(GstElement *elem, const gchar *p_name,
                                      guint p_value) {
  g_object_set(G_OBJECT(elem), p_name, p_value, NULL);
}

extern "C" void MCU_gst_elem_set_bool(GstElement *elem, const gchar *p_name,
                                      gboolean p_value) {
  g_object_set(G_OBJECT(elem), p_name, p_value, NULL);
}

extern "C" void MCU_gst_elem_set_caps(GstElement *elem, const gchar *p_name,
                                      const GstCaps *p_value) {
  g_object_set(G_OBJECT(elem), p_name, p_value, NULL);
}

extern "C" void MCU_gst_elem_set_structure(GstElement *elem,
                                           const gchar *p_name,
                                           const GstStructure *p_value) {
  g_object_set(G_OBJECT(elem), p_name, p_value, NULL);
}

