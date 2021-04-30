# FROM ubuntu:bionic
#FROM praqma/network-multitool
FROM alpine:3.13.5

ENV ISTIOD_ADDR="istiod.istio-system:15010"
ENV NAMESPACE="istio-system"
ENV ISTIO_CONFIG_STORE_SECRET=""

COPY aeraki /usr/local/bin/
ENTRYPOINT /usr/local/bin/aeraki -istiod-address=$ISTIOD_ADDR  -namespace=$NAMESPACE -config-store-secret=$ISTIO_CONFIG_STORE_SECRET