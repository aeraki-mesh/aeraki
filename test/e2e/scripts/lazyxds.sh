#!/usr/bin/env bash

set -ex
BASEDIR=$(dirname "$0")

TMP_DIR=$BASEDIR/../tmp

if [ -z "$BUILD_TAG" ]
then
  export BUILD_TAG=`git log --format="%H" -n 1`
fi

rm -rf $TMP_DIR
mkdir $TMP_DIR
envsubst < $BASEDIR/../common/lazyxds-controller.yaml > $TMP_DIR/lazyxds-controller-tmp.yaml
kubectl apply -f $TMP_DIR/lazyxds-controller-tmp.yaml
kubectl apply -f $BASEDIR/../../../lazyxds/install/lazyxds-egress.yaml
rm -rf $TMP_DIR
