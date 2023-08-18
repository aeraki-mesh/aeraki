#!/bin/bash

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

set -ex

BASEDIR=$(dirname "$0")/../../..

COMMON_DIR=$BASEDIR/test/e2e/common

if [ -z "$ISTIO_NAMESPACE" ]; then
  export ISTIO_NAMESPACE="istio-system"
fi

if [ -z "$ISTIO_VERSION" ]; then
  export ISTIO_VERSION=1.18.1
fi

rm -rf ~/.aeraki/istio/istio-config.yaml
mkdir -p ~/.aeraki/istio
envsubst <${COMMON_DIR}/istio-config.yaml> ~/.aeraki/istio/istio-config.yaml

[ -n $(istioctl version --remote=false |grep $ISTIO_VERSION) ] || (curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$ISTIO_VERSION  sh - && sudo mv $PWD/istio-$ISTIO_VERSION/bin/istioctl /usr/local/bin/)

istioctl x uninstall -f ~/.aeraki/istio/istio-config.yaml -y || true

# By default, istioctl will generates validatingwebhookconfigurations and mutatingwebhookconfigurations for istio-system.
# This is a known bug, so manually delete it here.
kubectl delete validatingwebhookconfigurations istiod-default-validator || true
kubectl delete mutatingwebhookconfigurations istio-revision-tag-default || true
kubectl delete mutatingwebhookconfigurations istio-revision-tag-default-${ISTIO_NAMESPACE} || true

kubectl delete ns ${ISTIO_NAMESPACE} || true
