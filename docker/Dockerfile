# Copyright Aeraki Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# FROM ubuntu:bionic
# FROM praqma/network-multitool
FROM alpine:3.13.5
ENV AERAKI_ISTIOD_ADDR="istiod.istio-system:15010"
ENV AERAKI_NAMESPACE="istio-system"
ENV AERAKI_ISTIO_CONFIG_STORE_SECRET=""
ENV AERAKI_XDS_LISTEN_ADDR=":15010"
ENV AERAKI_LOG_LEVEL="all:info"
ENV AERAKI_SERVER_ID=""
ENV AERAKI_CLUSTER_ID=""
ENV AERAKI_IS_MASTER="true"
ENV AERAKI_ENABLE_ENVOY_FILTER_NS_SCOPE="false"


COPY aeraki /usr/local/bin/
ENTRYPOINT /usr/local/bin/aeraki                                                     \
           -istiod-address=$AERAKI_ISTIOD_ADDR                                       \
           -root-namespace=$AERAKI_NAMESPACE                                              \
           -cluster-id=$AERAKI_CLUSTER_ID                                            \
           -config-store-secret=$AERAKI_ISTIO_CONFIG_STORE_SECRET                    \
           -xds-listen-address=$AERAKI_XDS_LISTEN_ADDR                               \
           -log-level=$AERAKI_LOG_LEVEL                                              \
           -server-id=$AERAKI_SERVER_ID                                              \
           -master=$AERAKI_IS_MASTER                                                 \
           -enable-envoy-filter-namespace-scope=$AERAKI_ENABLE_ENVOY_FILTER_NS_SCOPE \
