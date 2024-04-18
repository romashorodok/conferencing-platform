# https://mesonbuild.com/Builtin-options.html
# Working with meson 1.3.2 and python3.9 in venv
project('media-server', 'c', 'cpp',
  default_options: ['cpp_std=c++17', 'buildtype=debug'],
  version: '0.0.0',
  )

# cmake = import('cmake')
# opt_var = cmake.subproject_options()
# opt_var.add_cmake_defines({
#   'WITH_OPENCL': false,
#   'ENABLE_PRECOMPILED_HEADERS': false,
#   })
# opencv4 = cmake.subproject('opencv',options: opt_var )
# opencv4 = dependency('opencv')

cc = meson.get_compiler('c')
std = cc.find_library('stdc++', required: true)

media_server_mcu_proj = subproject('media-server-mcu')
media_server_mcu_lib = media_server_mcu_proj.get_variable('media_server_mcu_lib')

# pipelines_proj = subproject('pipelines')
# rtpvp8_lib = pipelines_proj.get_variable('rtpvp8_lib')
# pipeline_lib = pipelines_proj.get_variable('pipelines_lib')

go = find_program('go')

cmd_path = join_paths(meson.current_source_dir(), 'media-server/cmd/media-server')
# pkg_config_path = join_paths(meson.current_build_dir(), 'meson-uninstalled')

# pkg_config_path = '@0@:@1@'.format(join_paths(meson.current_build_dir(), 'meson-uninstalled'),'/usr/local/Cellar/opencv/4.9.0_7/lib/pkgconfig/')
pkg_config_path = '@0@:@1@'.format(
  join_paths(meson.current_build_dir(), 'meson-uninstalled'),
  join_paths(meson.current_build_dir(), 'build/unix-install'),
)

uninstalled_env = environment()
uninstalled_env.set('PKG_CONFIG_PATH', pkg_config_path)
uninstalled_deps = 'opencv4'

r = run_command('pkg-config', '--cflags', uninstalled_deps, check: true, env: uninstalled_env)
pkg_config_cflags = r.stdout().strip()

r = run_command('pkg-config', '--libs', uninstalled_deps, check: true, env: uninstalled_env)
pkg_config_ldflags = r.stdout().strip() + ' -lstdc++'
cgo_config_ldflags = '-w -s '

goenv = environment()
goenv.set('PKG_CONFIG_PATH', pkg_config_path)
goenv.set('CGO_ENABLED', '1')

# Go unable detect changes from C. By default it's cache C lib. By that behaviour compiler can't detect changes in cgo and pass cached code. To build with new cpp lib I must compile with `go build -a`.
# go_cache_path = join_paths(meson.current_build_dir(), 'go-build')
# goenv.set('GOCACHE', go_cache_path)

# goenv.set('GODEBUG', 'gocachehash=1')

goenv.set('CGO_LDFLAGS', cgo_config_ldflags + pkg_config_ldflags)
goenv.set('CGO_CFLAGS', pkg_config_cflags)

go_mod = custom_target('golang',
  output: 'media-server',
  env: goenv,
  depends: [media_server_mcu_lib],
  command: [go, 'build', 
    # '-a',
    '-o', '@OUTPUT@', cmd_path],
  install: true,
  install_dir: '',
  build_always_stale: true,
  build_by_default: true,
  )
