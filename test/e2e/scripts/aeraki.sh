#!/usr/bin/env bash

set -ex
export NAMESPACE="istio-system"
export ISTIOD_ADDR="istiod.istio-system:15010"

BASEDIR=$(dirname "$0")

if [ -z "$BUILD_TAG" ]
then
  export BUILD_TAG=`git log --format="%H" -n 1`
fi

envsubst < $BASEDIR/../common/aeraki.yaml > aeraki.yaml
kubectl create ns istio-system
kubectl apply -f $BASEDIR/../../../crd/kubernetes/customresourcedefinitions.gen.yaml
kubectl apply -f aeraki.yaml -n ${NAMESPACE}
