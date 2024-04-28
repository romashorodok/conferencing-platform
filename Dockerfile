FROM alpine:3.19.1 as media-server

RUN apk add --no-cache git meson curl
RUN apk add --no-cache pkgconfig ninja-build cmake make
RUN apk add --no-cache g++ musl-dev gcompat libstdc++ libffi-dev flex bison nasm

RUN apk add --no-cache opencv-dev
RUN apk add go --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community

WORKDIR /app

COPY meson.build .

COPY subprojects/gstreamer-1.0.wrap subprojects/gstreamer-1.0.wrap
RUN meson subprojects download gstreamer-1.0

COPY subprojects/media-server-mcu subprojects/media-server-mcu

COPY Makefile .
RUN make setup
RUN meson compile -C builddir

RUN --mount=type=cache,target=/go \
    --mount=type=bind,source=./pkg/go.mod,target=go.mod \
    go mod download -x

RUN --mount=type=cache,target=/go \
    --mount=type=bind,source=./media-server/go.mod,target=go.mod \
    go mod download -x

COPY build.sh .
RUN chmod +x build.sh

RUN go env -w GOCACHE=/go-cache

RUN --mount=type=cache,target=/go \
    --mount=type=cache,target=/go-cache \
    --mount=type=bind,source=./go.work,target=go.work \
    --mount=type=bind,source=./go.mod,target=go.mod \
    --mount=type=bind,source=./pkg,target=pkg \
    --mount=type=bind,source=./media-server,target=media-server,readonly \
    ./build.sh

FROM alpine:3.19.1 as media-server-bin
RUN apk add --no-cache libopencv_core libopencv_imgproc libintl
COPY --from=media-server /app/bin/media-server /app/media-server
# libopencv_aruco libopencv_calib3d libopencv_core libopencv_dnn libopencv_face libopencv_features2d libopencv_flann libopencv_highgui libopencv_imgcodecs libopencv_imgproc libopencv_ml libopencv_objdetect libopencv_optflow libopencv_photo libopencv_plot libopencv_shape libopencv_stitching libopencv_superres libopencv_tracking libopencv_video libopencv_videoio libopencv_videostab libopencv_ximgproc
# ENTRYPOINT [ "/app/media-server" ]

FROM nginx:alpine3.19 as gateway
RUN apk add --no-cache openssl bash envsubst certbot certbot-nginx

# TODO: standalone js build
RUN apk add --no-cache nodejs npm
WORKDIR /app

COPY ./client client
WORKDIR /app/client
RUN npm install

ARG DOMAIN=localhost
ARG VITE_MEDIA_SERVER=http://localhost/api
ARG VITE_MEDIA_SERVER_WS=wss://localhost/api
ARG VITE_MEDIA_SERVER_STUN=stun:stun.l.google.com:19302

ENV DOMAIN=${DOMAIN}
ENV VITE_MEDIA_SERVER=${VITE_MEDIA_SERVER}
ENV VITE_MEDIA_SERVER_WS=${VITE_MEDIA_SERVER_WS}
ENV VITE_MEDIA_SERVER_STUN=${VITE_MEDIA_SERVER_STUN}

RUN npm run build
RUN rm -fr node_modules

WORKDIR /app
COPY nginx.templ.conf .
COPY certs.sh . 

CMD ["bash", "-c", "chmod +x /app/certs.sh && source /app/certs.sh && echo $SSL_CERTIFICATE && envsubst '$DOMAIN $SSL_CERTIFICATE $SSL_CERTIFICATE_KEY' < /app/nginx.templ.conf > /app/nginx.conf && cat /app/nginx.conf && nginx -c /app/nginx.conf -g 'daemon off;'"]
