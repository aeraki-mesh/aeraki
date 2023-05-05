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

set -e

BASEDIR=$(dirname "$0")/../../..

if [ "$1" == "mode=tcm" ]; then
  if [ -z "$AERAKI_TAG" ]; then
    echo "failed to install aeraki for tcm, please set AERAKI_TAG environment variable"
    exit 1
  fi
  if [ -z "$AERAKI_ISTIOD_ADDR" ]; then
    echo "failed to install aeraki for tcm, please set AERAKI_ISTIOD_ADDR environment variable"
    exit 1
  fi
  if [ -z "$AERAKI_CLUSTER_ID" ]; then
    echo "failed to install aeraki for tcm, please set AERAKI_CLUSTER_ID environment variable"
    exit 1
  fi
fi

set -x

if [ -z "$AERAKI_TAG" ]; then
  export AERAKI_TAG=`git log --format="%H" -n 1`
fi

if [ -z "$ISTIO_NAMESPACE" ]; then
  export ISTIO_NAMESPACE="istio-system"
fi

if [ -z "$AERAKI_NAMESPACE" ]; then
  export AERAKI_NAMESPACE=${ISTIO_NAMESPACE}
fi

if [ -z "$AERAKI_ISTIOD_ADDR" ]; then
  export AERAKI_ISTIOD_ADDR="istiod.${ISTIO_NAMESPACE}:15010"
fi

if [ -z "$AERAKI_IS_MASTER" ]; then
  export AERAKI_IS_MASTER="true"
fi

if [ -z "$AERAKI_ENABLE_ENVOY_FILTER_NS_SCOPE" ]; then
  export AERAKI_ENABLE_ENVOY_FILTER_NS_SCOPE="false"
fi

if [ -z "$AERAKI_IMG_PULL_POLICY" ]; then
  export AERAKI_IMG_PULL_POLICY=Always
fi

if [ -z "$AERAKI_IMAGE" ]; then
  export AERAKI_IMAGE="ghcr.io/aeraki-mesh/aeraki"
fi

if [ -z "$AERAKI_XDS_ADDR" ]; then
  export AERAKI_XDS_ADDR="aeraki.${ISTIO_NAMESPACE}"
fi

if [ -z "$AERAKI_XDS_PORT" ]; then
  export AERAKI_XDS_PORT=":15010"
fi

kubectl create ns ${ISTIO_NAMESPACE} || true
kubectl create ns ${AERAKI_NAMESPACE} || true

rm -rf ~/.aeraki/aeraki.yaml
mkdir -p ~/.aeraki
envsubst < $BASEDIR/k8s/aeraki.yaml > ~/.aeraki/aeraki.yaml
if [ "$1" == "mode=tcm" ]; then
  kubectl apply -f $BASEDIR/k8s/tcm-apiservice.yaml
  kubectl apply -f $BASEDIR/k8s/tcm-istio-cm.yaml
else
  # ApplicationProtocol is changed from namespace scope to cluster scope
  kubectl delete crd applicationprotocols.metaprotocol.aeraki.io || true
  kubectl apply -f $BASEDIR/k8s/crd.yaml
fi
kubectl apply -f ~/.aeraki/aeraki.yaml -n ${AERAKI_NAMESPACE}
