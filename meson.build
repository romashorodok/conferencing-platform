# https://mesonbuild.com/Builtin-options.html

project('media-server', 'c', 'cpp',
  default_options: ['cpp_std=c++17', 'buildtype=debug'],
  version: '0.0.0',
  )

cc = meson.get_compiler('c')
std = cc.find_library('stdc++', required: true)

pipelines_proj = subproject('pipelines')
rtpvp8_dep = pipelines_proj.get_variable('rtpvp8_lib')
pipeline_dep = pipelines_proj.get_variable('pipelines_lib') 

go = find_program('go')

cmd_path = join_paths(meson.current_source_dir(), 'media-server/cmd/media-server')
pkg_config_path = join_paths(meson.current_build_dir(), 'meson-uninstalled')
go_cache_path = join_paths(meson.current_build_dir(), 'go-build')

goenv = environment()
goenv.set('PKG_CONFIG_PATH', pkg_config_path)
goenv.set('GOCACHE', go_cache_path)

go_mod = custom_target('golang',
  output: 'media-server',
  env: goenv,
  depends: [rtpvp8_dep, pipeline_dep],
  command: [go, 'build', '-o', '@OUTPUT@', cmd_path],
  install: true,
  install_dir: '',
  build_always_stale: true,
  )
