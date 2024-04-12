
gstreamer_full_dep = dependency('gstreamer-full-1.0', fallback: 'gstreamer-1.0')
gstreamer_base_dep = dependency('gstreamer-base-1.0')
gstreamer_app_dep = dependency('gstreamer-app-1.0')
gstreamer_opencv_dep = dependency('gstopencv')
opencv_dep = dependency('opencv4')

media_server_mcu_sources = [
  'main.cpp',

  'vision/visioncannyfilter.cpp',
  'vision/visionframe.cpp',
  'vision/visionutils.cpp',
]

media_server_mcu_headers = [
  'main.h',

  'vision/visioncannyfilter.h',
  'vision/visionframe.h',
  'vision/visionutils.h',
]

media_server_mcu_lib = library('media-server-mcu',
 media_server_mcu_sources + media_server_mcu_headers,
 dependencies: [
   gstreamer_full_dep, 
   gstreamer_app_dep,
   opencv_dep, 
   gstreamer_opencv_dep,
  ],
)

pkgconfig.generate(media_server_mcu_lib)

