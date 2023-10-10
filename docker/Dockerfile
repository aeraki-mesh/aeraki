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

FROM alpine:3.17

LABEL org.opencontainers.image.source="https://github.com/aeraki-mesh/aeraki"
LABEL org.opencontainers.image.authors="zhaohuabing@gmail.com"
LABEL org.opencontainers.image.documentation="https://aeraki.net"

ARG AERAKI_ROOT_BIN_DIR
ARG ARCH
ARG OS

RUN apk update && \
    apk add curl

COPY ${AERAKI_ROOT_BIN_DIR}/${ARCH}/${OS}/aeraki /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/aeraki"]
