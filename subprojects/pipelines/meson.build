project('pipelines-1.0', 'cpp', default_options: ['cpp_std=c++17'])

api_version = '1.0'

pkgconfig = import('pkgconfig')
pipelines = get_option('pipelines')

if pipelines.enabled()
  gstreamer_full_dep = dependency('gstreamer-full-1.0', fallback: 'gstreamer-1.0', required: pipelines)
  gstreamer_base_dep = dependency('gstreamer-base-1.0', required: pipelines)
  gstreamer_app_dep = dependency('gstreamer-app-1.0', required: pipelines)
endif 

pipelines_inc = include_directories('.')
subdir('pipelines')
