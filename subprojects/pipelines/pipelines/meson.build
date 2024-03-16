
gstreamer_full_dep = dependency('gstreamer-full-1.0', fallback: 'gstreamer-1.0', required: pipelines)

gstreamer_base_dep = dependency('gstreamer-base-1.0', required: pipelines)

gstreamer_app_dep = dependency('gstreamer-app-1.0', required: pipelines)
gstreamer_opencv_dep = dependency('gstopencv', required: pipelines)

opencv_dep = dependency('opencv4')

subdir('rtpvp8')

pipelines_sources = [
  'pipelines.cpp',
  'dummy/dummytransform.cpp',

  'vision/visioncannyfilter.cpp',
  'vision/visionframe.cpp',
  'vision/visionutils.cpp',
]

pipelines_headers = [
  'pipelines.h',
  'dummy/dummyelements.h',
  'dummy/dummytransform.h',

  'vision/visioncannyfilter.h',
  'vision/visionframe.h',
  'vision/visionutils.h',
]

if pipelines.enabled()
  pipelines_lib = library('pipelines-@0@'.format(api_version),
    pipelines_sources + pipelines_headers,
    dependencies: [gstreamer_full_dep, gstreamer_app_dep, rtpvp8_dep, opencv_dep, gstreamer_opencv_dep],
  )

  pipelines_dep = declare_dependency(link_whole: pipelines_lib,
    include_directories: [pipelines_inc, rtpvp8_inc],
    dependencies: [gstreamer_full_dep, gstreamer_app_dep, rtpvp8_dep, opencv_dep, gstreamer_opencv_dep],
  )

  pkgconfig.generate(pipelines_lib) 
  meson.override_dependency(pipelines_lib.name(), pipelines_dep)
endif
