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

SCRIPTS_DIR=$BASEDIR/test/e2e/scripts

MODE=$1

if [ -z "$ISTIO_NAMESPACE" ]; then
  export ISTIO_NAMESPACE="istio-system"
fi

if [ -z "$AERAKI_NAMESPACE" ]; then
  export AERAKI_NAMESPACE=${ISTIO_NAMESPACE}
fi

if [ "${MODE}" == "tcm" ]; then
  kubectl delete -f $BASEDIR/k8s/tcm-apiservice.yaml
  kubectl delete -f $BASEDIR/k8s/tcm-istio-cm.yaml
else
  kubectl delete -f $BASEDIR/k8s/aeraki.yaml
  kubectl delete -f $BASEDIR/k8s/crd.yaml
fi

kubectl delete validatingwebhookconfigurations aeraki-${AERAKI_NAMESPACE} || true

bash ${SCRIPTS_DIR}/remove-aeraki-configmap.sh
