#!/bin/sh

libs="opencv4 gstreamer-full-1.0 media-server-mcu"

pwd=$(pwd)
pkg_config_path=$pwd/builddir/meson-uninstalled:$pwd/build/unix-install

pkg_config_cflags=$(PKG_CONFIG_PATH=$pkg_config_path pkg-config --cflags $libs)
pkg_config_ldflags=$(PKG_CONFIG_PATH=$pkg_config_path pkg-config --libs $libs)

cgo_cflags="$pkg_config_cflags"
cgo_ldflags="$pkg_config_ldflags -lstdc++ -w -s"

media_server_cmd="$pwd/media-server/cmd/media-server"

echo $cgo_cflags $cgo_ldflags

PKG_CONFIG_PATH=$pkg_config_path \
    CGO_ENABLED=1 \
    CGO_CFLAGS="$cgo_cflags" \
    CGO_LDFLAGS="$cgo_ldflags" \
    go build -o media-server $media_server_cmd
