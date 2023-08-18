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

if [ -z "$AERAKI_TAG" ]; then
  export AERAKI_TAG="1.4.1"
fi
bash ${SCRIPTS_DIR}/istio.sh
bash ${SCRIPTS_DIR}/addons.sh
bash ${SCRIPTS_DIR}/aeraki.sh

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
