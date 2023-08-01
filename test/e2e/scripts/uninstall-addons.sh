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

if [ -z "$ISTIO_NAMESPACE" ]; then
  export ISTIO_NAMESPACE="istio-system"
fi

rm -rf ~/.aeraki/istio/addons
mkdir -p ~/.aeraki/istio/addons
envsubst < $BASEDIR/demo/addons/grafana.yaml > ~/.aeraki/istio/addons/grafana.yaml
envsubst < $BASEDIR/demo/addons/jaeger.yaml > ~/.aeraki/istio/addons/jaeger.yaml
envsubst < $BASEDIR/demo/addons/prometheus.yaml > ~/.aeraki/istio/addons/prometheus.yaml
kubectl delete -f ~/.aeraki/istio/addons/grafana.yaml
kubectl delete -f ~/.aeraki/istio/addons/jaeger.yaml
kubectl delete -f ~/.aeraki/istio/addons/prometheus.yaml

rm -rf ~/.aeraki/istio/gateway
mkdir -p ~/.aeraki/istio/gateway
envsubst < $BASEDIR/demo/gateway/demo-ingress.yaml > ~/.aeraki/istio/gateway/demo-ingress.yaml
envsubst < $BASEDIR/demo/gateway/istio-ingressgateway.yaml > ~/.aeraki/istio/gateway/istio-ingressgateway.yaml
kubectl delete -f ~/.aeraki/istio/gateway/demo-ingress.yaml
kubectl delete -f ~/.aeraki/istio/gateway/istio-ingressgateway.yaml
