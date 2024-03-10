# https://mesonbuild.com/Builtin-options.html

project('media-server', 'c', 'cpp',
  default_options: ['cpp_std=c++17', 'buildtype=debug'],
  version: '0.0.0',
  )

cc = meson.get_compiler('c')
std = cc.find_library('stdc++', required: true)

pipelines = get_option('pipelines')

gstreamer_full_dep = dependency('gstreamer-full-1.0', fallback: 'gstreamer-1.0', required: pipelines)

go = find_program('go')

cmd_path = join_paths(meson.current_source_dir(), 'cmd/media-server')
pkg_config_path = join_paths(meson.current_build_dir(), 'meson-uninstalled')

goenv = environment()
goenv.set('PKG_CONFIG_PATH', pkg_config_path)

go_mod = custom_target('golang',
  output: 'media-server',
  env: goenv,
  command: [go, 'build', '-a', '-o', '@OUTPUT@', cmd_path],
  install: true,
  install_dir: '',
  build_always_stale: true,
  )
