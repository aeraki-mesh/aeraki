#!/usr/bin/env bash

set -ex

BASEDIR=$(dirname "$0")

if [ -z "$BUILD_TAG" ]
then
  export BUILD_TAG=`git log --format="%H" -n 1`
fi
sed "s/\${BUILD_TAG}/$BUILD_TAG/" $BASEDIR/../common/aeraki.yaml > aeraki.yaml
kubectl create ns istio-system
kubectl apply -f aeraki.yaml -n istio-system
