
rtpvp8_sources = [
  'rtpvp8.cpp',
]

rtpvp8_headers = [
  'rtpvp8.h',
]

rtpvp8_inc = include_directories('.')

if pipelines.enabled()
  rtpvp8_lib = static_library('rtpvp8-@0@'.format(api_version), 
    rtpvp8_sources + rtpvp8_headers,
    include_directories: [pipelines_inc, rtpvp8_inc],
    dependencies: [gstreamer_full_dep, gstreamer_base_dep, gstreamer_app_dep, opencv_dep],
  )

  rtpvp8_dep = declare_dependency(link_with: rtpvp8_lib,
    include_directories: [pipelines_inc, rtpvp8_inc],
    dependencies: [gstreamer_full_dep, gstreamer_base_dep, gstreamer_app_dep, opencv_dep],
  )

  pkgconfig.generate(rtpvp8_lib)
  meson.override_dependency(rtpvp8_lib.name(), rtpvp8_dep)
endif
