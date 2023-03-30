#!/bin/bash

# This script is copied from github.com/apache/skywalking
# ----------------------------------------------------------------------------
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
# ----------------------------------------------------------------------------

set -ex

BASEDIR=$(dirname "$0")

COMMON_DIR=$BASEDIR/../../../test/e2e/common

if [ -z "$ISTIO_NAMESPACE" ]; then
  export ISTIO_NAMESPACE="istio-system"
fi

if [ -z "$ISTIO_VERSION" ]; then
  export ISTIO_VERSION=1.14.5
fi

if [ -z "$AERAKI_TAG" ]; then
  export AERAKI_TAG=1.2.3
fi

mkdir -p ~/.aeraki/istio
envsubst < ${COMMON_DIR}/istio-config.yaml > ~/.aeraki/istio/istio-config.yaml

[ -n "$(istioctl version --remote=false |grep $ISTIO_VERSION)" ] || (curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$ISTIO_VERSION  sh - && sudo mv $PWD/istio-$ISTIO_VERSION/bin/istioctl /usr/local/bin/)

/usr/local/bin/istioctl install -f ~/.aeraki/istio/istio-config.yaml -y

kubectl delete ns ${ISTIO_NAMESPACE}