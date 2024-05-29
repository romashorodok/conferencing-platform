FROM golang:alpine3.20 as media-server

WORKDIR /app

RUN go env -w GOCACHE=/go-cache

RUN --mount=type=cache,target=/go \
    --mount=type=bind,source=./pkg/go.mod,target=go.mod \
    go mod download -x

RUN --mount=type=cache,target=/go \
    --mount=type=bind,source=./media-server/go.mod,target=go.mod \
    go mod download -x

RUN --mount=type=cache,target=/go \
    --mount=type=cache,target=/go-cache \
    --mount=type=bind,source=./go.work,target=go.work \
    --mount=type=bind,source=./go.mod,target=go.mod \
    --mount=type=bind,source=./pkg,target=pkg \
    --mount=type=bind,source=./media-server,target=media-server,readonly \
    CGO_ENABLED=0 GOOS=linux go build -o bin/media-server ./media-server/cmd/media-server

FROM alpine:3.20 as media-server-bin
COPY --from=media-server /app/bin/media-server /app/media-server

FROM nginx:alpine3.19 as gateway
RUN apk add --no-cache openssl bash envsubst certbot certbot-nginx

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

ENV BASE64_FULL_CHAIN=
ENV BASE64_PRIV_KEY=

CMD ["bash", "-c", "chmod +x /app/certs.sh && source /app/certs.sh && echo $SSL_CERTIFICATE && envsubst '$DOMAIN $SSL_CERTIFICATE $SSL_CERTIFICATE_KEY' < /app/nginx.templ.conf > /app/nginx.conf && cat /app/nginx.conf && nginx -c /app/nginx.conf -g 'daemon off;'"]
