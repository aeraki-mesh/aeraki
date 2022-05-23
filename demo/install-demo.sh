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

BASEDIR=$(dirname "$0")/..

DEMO=$1

SCRIPTS_DIR=$BASEDIR/test/e2e/scripts
COMMON_DIR=$BASEDIR/test/e2e/common
export ISTIO_VERSION=1.12.7
export AERAKI_TAG=1.1.0

bash ${SCRIPTS_DIR}/istio.sh -y -f ${COMMON_DIR}/istio-config.yaml
bash ${SCRIPTS_DIR}/aeraki.sh

kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.10/samples/addons/prometheus.yaml -n istio-system
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.10/samples/addons/grafana.yaml -n istio-system

kubectl create namespace kiali-operator
helm install \
    --set cr.create=true \
    --set cr.namespace=istio-system \
    --namespace kiali-operator \
    --repo https://kiali.org/helm-charts \
    kiali-operator \
    kiali-operator

kubectl apply -f $BASEDIR/demo/gateway/demo-ingress.yaml -n istio-system
kubectl apply -f $BASEDIR/demo/gateway/istio-ingressgateway.yaml -n istio-system

if [ "${DEMO}" == "default" ]
then
    echo "install default demo"
    bash ${BASEDIR}/demo/metaprotocol-dubbo/install.sh
    bash ${BASEDIR}/demo/metaprotocol-thrift/install.sh
elif [ "${DEMO}" == "brpc" ]
then
    echo "install brpc demo"
    bash ${BASEDIR}/demo/metaprotocol-brpc/install.sh
elif [ "${DEMO}" == "kafka" ]
then
    echo "install kafka demo"
    bash ${BASEDIR}/demo/kafka/install.sh
fi
