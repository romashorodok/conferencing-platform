
setup:
	./venv/bin/meson setup --reconfigure -Dpkg_config_path=./build/unix-install/:./builddir/meson-uninstalled/ -Dpipelines:pipelines=enabled --default-library=static --force-fallback-for=gstreamer-1.0,coreelements,glib,libffi,pcre2,vpx,matroska,app,rtp,rtpmanager,opencv,opencv4 -Dauto_features=disabled -Dglib:tests=false -Djson-glib:tests=false -Dpcre2:test=false  -Dgstreamer-1.0:ugly=disabled -Dgstreamer-1.0:ges=disabled -Dgstreamer-1.0:devtools=disabled -Dgstreamer-1.0:default_library=static -Dgstreamer-1.0:rtsp_server=disabled -Dgstreamer-1.0:gst-full-target-type=static_library -Dgstreamer-1.0:gst-full-libraries=coreelements,app,video,audio,codecparsers,vpx,matroska,rtp,rtpmanager -Dgst-plugins-base:playback=enabled -Dgst-plugins-base:videotestsrc=enabled -Dgst-plugins-base:videoconvertscale=enabled  -Dgst-plugins-good:vpx=enabled -Dgst-plugins-good:matroska=enabled -Dgst-plugins-good:rtp=enabled -Dgst-plugins-good:rtpmanager=enabled  -Dgst-plugins-base:app=enabled -Dgst-plugins-bad:videoparsers=enabled -Dgst-plugins-base:typefind=enabled -Dvpx:vp8=enabled -Dvpx:vp9=enabled -Dvpx:vp8_encoder=enabled -Dvpx:vp8_decoder=enabled -Dvpx:vp9_encoder=enabled -Dvpx:vp9_decoder=enabled builddir

build:
	meson compile -C builddir && ./builddir/media-server

opencv-setup:
	meson subprojects download opencv && mkdir build 

# -DCMAKE_INSTALL_PREFIX=../builddir/meson-private 
# -DCMAKE_INSTALL_PREFIX=../build 
opencv:
	 cd build && \
		cmake \
		-GNinja \
		-DOPENCV_GENERATE_PKGCONFIG=ON \
		-DBUILD_SHARED_LIBS=OFF \
		-DBUILD_TESTS=OFF \
		-DBUILD_PERF_TESTS=OFF \
		-DBUILD_EXAMPLES=OFF \
		-DBUILD_opencv_apps=OFF \
		-DBUILD_opencv_python3=OFF \
		-DBUILD_JAVA=OFF  \
		-DWITH_GSTREAMER=OFF \
		-DWITH_ITT=OFF \
		../subprojects/opencv && ninja

download:
	meson subprojects download gstreamer-1.0

sdk:
	sudo xcode-select -switch /Library/Developer/CommandLineTools
