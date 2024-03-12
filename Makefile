
setup:
	./venv/bin/meson setup --reconfigure -Dpipelines:pipelines=enabled --default-library=static --force-fallback-for=gstreamer-1.0,coreelements,glib,libffi,pcre2,vpx,matroska,app,rtp,rtpmanager -Dauto_features=disabled -Dglib:tests=false -Djson-glib:tests=false -Dpcre2:test=false  -Dgstreamer-1.0:ugly=disabled -Dgstreamer-1.0:ges=disabled -Dgstreamer-1.0:devtools=disabled -Dgstreamer-1.0:default_library=static -Dgstreamer-1.0:rtsp_server=disabled -Dgstreamer-1.0:gst-full-target-type=static_library -Dgstreamer-1.0:gst-full-libraries=coreelements,app,video,audio,codecparsers,vpx,matroska,rtp,rtpmanager -Dgst-plugins-base:playback=enabled -Dgst-plugins-base:videotestsrc=enabled  -Dgst-plugins-good:vpx=enabled -Dgst-plugins-good:matroska=enabled -Dgst-plugins-good:rtp=enabled -Dgst-plugins-good:rtpmanager=enabled  -Dgst-plugins-base:app=enabled -Dgst-plugins-bad:videoparsers=enabled -Dgst-plugins-base:typefind=enabled -Dvpx:vp8=enabled -Dvpx:vp9=enabled -Dvpx:vp8_encoder=enabled -Dvpx:vp8_decoder=enabled -Dvpx:vp9_encoder=enabled -Dvpx:vp9_decoder=enabled builddir

download:
	meson subprojects download gstreamer-1.0
