# FROM ubuntu:bionic
FROM praqma/network-multitool

ENV istiodaddr="istiod.istio-system:15012"
ENV listeningAddress=":1109"

COPY aeraki /usr/local/bin/
ENTRYPOINT /usr/local/bin/aeraki -istiodaddr=$istiodaddr -listeningAddress=$listeningAddress