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

if [ -z "$AERAKI_CONFIGMAP_NAME" ]; then
  export AERAKI_CONFIGMAP_NAME=aeraki-bootstrap-config
fi

namespaces=`kubectl get ns | awk 'NR != 1 {print $1}'`

for ns in  $namespaces; do
  kubectl delete configmap $AERAKI_CONFIGMAP_NAME -n $ns || true
done
