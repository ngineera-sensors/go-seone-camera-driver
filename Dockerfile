FROM golang:1.19.4-alpine3.16
WORKDIR /var/lib/app

# RUN apk --update --upgrade --no-cache add \
#       bash \
#       alpine-sdk \
#       python3 \
#       py3-pip \
#       py3-yaml \
#       py3-ply \
#       py3-jinja2 \
#       g++ \
#       meson \
#       ninja \
#       pkgconfig \
#       boost \
#       openssl \
#       libevent \
#       raspberrypi \
#       cmake && \
#     apk --no-cache add \
#       eudev-dev \
#       boost-dev \
#       gnutls-dev \
#       openssl-dev \
#       libevent-dev \
#       v4l-utils-dev \
#       raspberrypi-dev \
#       linux-headers && \
#     cd /opt && \
#     git clone https://git.libcamera.org/libcamera/libcamera.git --depth=1 && \
#     cd libcamera && \
#     meson build && \
#     ninja -C build install && \
#     cd / && \
#     rm -rf /tmp/* /var/tmp/* ~/.cache
# RUN libcamera-hello --list-cameras

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./fspdriver ./fspdriver
COPY ./main.go ./

COPY ./include/* /usr/local/include/
COPY ./lib/* /usr/local/lib/

# RUN go build -o app

# CMD ./app
