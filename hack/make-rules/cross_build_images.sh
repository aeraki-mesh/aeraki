#!/usr/bin/env bash

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

set -o errexit
set -o nounset
set -o pipefail

AERAKI_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
AERAKI_ROOT_BIN_DIR=out
AERAKI_DOCKERFILE_PATH=${AERAKI_ROOT}/docker
AERAKI_MAIN_PATH=${AERAKI_ROOT}/"${AERAKI_MAIN_PATH:-/cmd/aeraki/main.go}"
AERAKI_IMAGE_COMMIT_TAG=$(git rev-parse "HEAD^{commit}")
AERAKI_IMAG_TAG="${AERAKI_IMAG_TAG:-latest}"
AERAKI_REPO="${AERAKI_REPO:-ghcr.io/aeraki-mesh}"
AERAKI_IMAGE_REPO_NAME="${IMAGE_REPO_NAME:-aeraki}"
AERAKI_REL_OSARCH="${IMAGE_REPO_NAME:-amd64 arm64}"
AERAKI_REL_OS="${AERAKI_REL_OS:-linux}"

function clean() {
  rm -rf ${AERAKI_ROOT}/${AERAKI_ROOT_BIN_DIR}
}

function build_bin() {
    for ARCH in ${AERAKI_REL_OSARCH}; do \
        for OS in ${AERAKI_REL_OS}; do \
            CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} go build -o=${AERAKI_ROOT_BIN_DIR}/${ARCH}/${OS}/${AERAKI_IMAGE_REPO_NAME} ${AERAKI_MAIN_PATH};\
        done
    done
    echo "build_bin successful"
}

function build_images() {
    for ARCH in ${AERAKI_REL_OSARCH}; do \
        for OS in ${AERAKI_REL_OS}; do \
            docker buildx build --build-arg AERAKI_ROOT_BIN_DIR=${AERAKI_ROOT_BIN_DIR} --build-arg ARCH=${ARCH} --build-arg OS=${OS} --no-cache --platform ${OS}/${ARCH} -t ${AERAKI_REPO}/${AERAKI_IMAGE_REPO_NAME}:${AERAKI_IMAG_TAG}-${OS}-${ARCH} -f ${AERAKI_DOCKERFILE_PATH}/Dockerfile . --load ;\
        done
    done
    echo "build_images successful"
}

function push_images() {
    for ARCH in ${AERAKI_REL_OSARCH}; do \
        for OS in ${AERAKI_REL_OS}; do \
            docker push ${AERAKI_REPO}/${AERAKI_IMAGE_REPO_NAME}:${AERAKI_IMAG_TAG}-${OS}-${ARCH};\
        done
    done
    echo "push_images successful"
}

function create_multi_arch_images() {
    for OS in ${AERAKI_REL_OS}; do \
        local IMAGES=""
        for ARCH in ${AERAKI_REL_OSARCH}; do \
            IMAGES="${IMAGES} ${AERAKI_REPO}/${AERAKI_IMAGE_REPO_NAME}:${AERAKI_IMAG_TAG}-${OS}-${ARCH}"
        done
        echo ${IMAGES}
        docker manifest create ${AERAKI_REPO}/${AERAKI_IMAGE_REPO_NAME}:${AERAKI_IMAG_TAG} ${IMAGES} --amend
        docker manifest push ${AERAKI_REPO}/${AERAKI_IMAGE_REPO_NAME}:${AERAKI_IMAG_TAG}
    done
    echo "create_multi_arch_images successful"
}

clean

build_bin

build_images

push_images

create_multi_arch_images
