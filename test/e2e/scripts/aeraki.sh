#!/bin/bash

set -e

BASEDIR=$(dirname "$0")

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

if [ -z "$AERAKI_ISTIOD_ADDR" ]; then
  export AERAKI_ISTIOD_ADDR="istiod.istio-system:15010"
fi

if [ -z "$AERAKI_NAMESPACE" ]; then
  export AERAKI_NAMESPACE="istio-system"
fi

if [ -z "$AERAKI_IS_MASTER" ]; then
  export AERAKI_IS_MASTER="true"
fi

if [ -z "$AERAKI_ENABLE_ENVOY_FILTER_NS_SCOPE" ]; then
  export AERAKI_ENABLE_ENVOY_FILTER_NS_SCOPE="false"
fi

mkdir -p ~/.aeraki
envsubst < $BASEDIR/../../../k8s/aeraki.yaml > ~/.aeraki/aeraki.yaml
if [ "$1" == "mode=tcm" ]; then
  kubectl apply -f $BASEDIR/../../../k8s/tcm-apiservice.yaml
else
  kubectl apply -f $BASEDIR/../../../k8s/crd.yaml
fi
kubectl apply -f ~/.aeraki/aeraki.yaml -n ${AERAKI_NAMESPACE}
