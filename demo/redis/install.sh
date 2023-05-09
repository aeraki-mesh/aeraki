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

set +x

if ! command -v jq &> /dev/null
then
    echo "can't find jq, please install jq first"
    exit 1
fi

set -x

BASEDIR=$(dirname "$0")
source $BASEDIR/../common_func.sh

kubectl create ns redis
LabelIstioInjectLabel redis
kubectl -n redis apply -f $BASEDIR/redis-client.yaml
kubectl -n redis apply -f $BASEDIR/redis-single.yaml
kubectl -n redis apply -f $BASEDIR/redis-cluster.yaml
kubectl -n redis wait deployment redis-single --for condition=Available=True --timeout=600s
kubectl -n redis rollout status --watch statefulset/redis-cluster --timeout=600s
kubectl -n redis wait pod --selector=app=redis-cluster --for=condition=ContainersReady=True --timeout=600s -o jsonpath='{.status.podIP}'
kubectl exec -it redis-cluster-0 -c redis -n redis -- redis-cli --cluster create --cluster-yes --cluster-replicas 1 $(kubectl get pod -n redis -l=app=redis-cluster -o json | jq -r '.items[] | .status.podIP + ":6379"')
