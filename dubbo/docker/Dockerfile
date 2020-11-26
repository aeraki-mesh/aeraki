# FROM ubuntu:bionic
FROM praqma/network-multitool

ENV istiodAddr="istiod.istio-system:15012"
ENV listeningAddr=":1109"

COPY aeraki-dubbo /usr/local/bin/
ENTRYPOINT /usr/local/bin/aeraki-dubbo -istiodaddr=$istiodAddr -listeningAddr=$listeningAddr