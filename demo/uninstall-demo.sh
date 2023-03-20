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

kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-1.14/samples/addons/prometheus.yaml
kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-1.14/samples/addons/grafana.yaml
kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-1.14/samples/addons/jeager.yaml

kubectl delete -f $BASEDIR/demo/gateway/demo-ingress.yaml -n istio-system
kubectl delete -f $BASEDIR/demo/gateway/istio-ingressgateway.yaml -n istio-system


if [ "${DEMO}" == "default" ]
then
    bash ${BASEDIR}/demo/metaprotocol-dubbo/uninstall.sh
    bash ${BASEDIR}/demo/metaprotocol-thrift/uninstall.sh
elif [ "${DEMO}" == "brpc" ]
then
    bash ${BASEDIR}/demo/metaprotocol-brpc/uninstall.sh
elif [ "${DEMO}" == "kafka" ]
then
    bash ${BASEDIR}/demo/kafka/uninstall.sh
fi

kubectl delete ns istio-system
