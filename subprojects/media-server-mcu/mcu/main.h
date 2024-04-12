#ifndef __MAIN_H__
#define __MAIN_H__

#include <gst/app/gstappsink.h>
#include <gst/gst.h>

#ifdef __cplusplus
extern "C" {
#endif

void MCU_setup();
void MCU_version();

GstBuffer *MCU_gst_buffer_new_wrapped(gchar *src, gsize len);

void MCU_gst_bin_add(GstElement *p, GstElement *element);

void MCU_gst_elem_set_string(GstElement *elem, const gchar *p_name,
                             gchar *p_value);
void MCU_gst_elem_set_int(GstElement *elem, const gchar *p_name, gint p_value);
void MCU_gst_elem_set_uint(GstElement *elem, const gchar *p_name,
                           guint p_value);
void MCU_gst_elem_set_bool(GstElement *elem, const gchar *p_name,
                           gboolean p_value);
void MCU_gst_elem_set_caps(GstElement *elem, const gchar *p_name,
                           const GstCaps *p_value);
void MCU_gst_elem_set_structure(GstElement *elem, const gchar *p_name,
                                const GstStructure *p_value);

#ifdef __cplusplus
}
#endif

#endif
