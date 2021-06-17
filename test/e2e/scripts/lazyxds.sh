#!/usr/bin/env bash

set -ex
BASEDIR=$(dirname "$0")

if [ -z "$BUILD_TAG" ]
then
  export BUILD_TAG=`git log --format="%H" -n 1`
fi

envsubst < $BASEDIR/../common/lazyxds-controller.yaml > lazyxds-controller.yaml
kubectl apply -f lazyxds-controller.yaml
kubectl apply -f $BASEDIR/../../../lazyxds/install/lazyxds-egress.yaml
