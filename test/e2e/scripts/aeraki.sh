#!/usr/bin/env bash

set -ex

export BUILD_TAG=`git log --format="%H" -n 1`
sed "s/\${BUILD_TAG}/$BUILD_TAG/" test/e2e/common/aeraki.yaml > aeraki.yaml
kubectl apply -f aeraki.yaml -n istio-system