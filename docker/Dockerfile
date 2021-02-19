# FROM ubuntu:bionic
FROM praqma/network-multitool

ENV istiodAddr="istiod.istio-system:15012"
ENV listeningAddr=":1109"

COPY aeraki /usr/local/bin/
ENTRYPOINT /usr/local/bin/aeraki -istiodaddr=$istiodAddr