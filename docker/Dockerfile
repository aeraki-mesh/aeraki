# FROM ubuntu:bionic
#FROM praqma/network-multitool
FROM alpine:3.13.5

ENV AERAKI_ISTIOD_ADDR="istiod.istio-system:15010"
ENV AERAKI_NAMESPACE="istio-system"
ENV AERAKI_ISTIO_CONFIG_STORE_SECRET=""
ENV AERAKI_XDS_LISTEN_ADDR=":15010"
ENV AERAKI_LOG_LEVEL="all:info"

COPY aeraki /usr/local/bin/
ENTRYPOINT /usr/local/bin/aeraki                                     \
           -istiod-address=$AERAKI_ISTIOD_ADDR                       \
           -namespace=$AERAKI_NAMESPACE                              \
           -config-store-secret=$AERAKI_ISTIO_CONFIG_STORE_SECRET    \
           -xds-listen-address=$AERAKI_XDS_LISTEN_ADDR                  \
           -log-level=$AERAKI_LOG_LEVEL