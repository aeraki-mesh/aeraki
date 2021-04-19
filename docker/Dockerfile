# FROM ubuntu:bionic
#FROM praqma/network-multitool
FROM alpine:3.13.5

ENV ISTIOD_ADDR="istiod.istio-system:15010"

COPY aeraki /usr/local/bin/
ENTRYPOINT /usr/local/bin/aeraki -istiodaddr=$ISTIOD_ADDR
