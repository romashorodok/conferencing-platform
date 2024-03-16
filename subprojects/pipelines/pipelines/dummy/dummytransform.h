#ifndef __GST_TYPE_DUMMY_TRANSFORM_H__
#define __GST_TYPE_DUMMY_TRANSFORM_H__

#include <glib.h>
#include <gst/base/gstbasetransform.h>
#include <gst/gst.h>

#ifdef BUILDING_GST_APP
#define GST_APP_API GST_API_EXPORT /* from config.h */
#else
#define GST_APP_API GST_API_IMPORT
#endif

G_BEGIN_DECLS

#define GST_TYPE_DUMMY_TRANSFORM (gst_dummy_transform_get_type())

typedef struct _GstDummyTransform GstDummyTransform;
typedef struct _GstDummyTransformClass GstDummyTransformClass;
typedef struct _GstDummyTransformPrivate GstDummyTransformPrivate;

#define GST_DUMMY_TRANSFORM(obj)                                               \
  (G_TYPE_CHECK_INSTANCE_CAST((obj), GST_TYPE_DUMMY_TRANSFORM, GstAppSink))
#define GST_DUMMY_TRANSFORM_CLASS(klass)                                       \
  (G_TYPE_CHECK_CLASS_CAST((klass), GST_TYPE_DUMMY_TRANSFORM, GstAppSinkClass))
#define GST_IS_DUMMY_TRANSFORM(obj)                                            \
  (G_TYPE_CHECK_INSTANCE_TYPE((obj), GST_TYPE_DUMMY_TRANSFORM))
#define GST_IS_DUMMY_TRANSFORM_CLASS(klass)                                    \
  (G_TYPE_CHECK_CLASS_TYPE((klass), GST_TYPE_DUMMY_TRANSFORM))
#define GST_DUMMY_TRANSFORM_CAST(obj) ((GstDummyTransform *)(obj))

// Represent name for registration of type. Needed for class prefixes.
#define DUMMY_TRANSFORM_NAME gst_dummy_transform
// Represent name real name of structure.
#define DUMMY_TRANSFORM_TYPE_NAME GstDummyTransform

struct _GstDummyTransform {
  GstBaseTransform basetransform;

  /*< private >*/
  GstDummyTransformPrivate *priv;
};

struct _GstDummyTransformClass {
  GstBaseTransformClass basetransform_class;
};

G_DEFINE_AUTOPTR_CLEANUP_FUNC(DUMMY_TRANSFORM_TYPE_NAME, gst_object_unref)

GType gst_dummy_transform_get_type(void);

G_END_DECLS

#endif /* __GST_TYPE_DUMMY_TRANSFORM_H__ */
