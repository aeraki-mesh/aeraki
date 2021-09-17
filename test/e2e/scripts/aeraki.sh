#!/usr/bin/env bash

set -ex
export NAMESPACE="istio-system"
export ISTIOD_ADDR="istiod.istio-system:15010"

BASEDIR=$(dirname "$0")

if [ -z "$BUILD_TAG" ]
then
  export BUILD_TAG=`git log --format="%H" -n 1`
fi

envsubst < $BASEDIR/../../../k8s/aeraki.yaml > aeraki.yaml
kubectl apply -f $BASEDIR/../../../k8s/crd.yaml
kubectl apply -f aeraki.yaml -n ${NAMESPACE}
