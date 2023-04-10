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

# Go parameters
GOCMD?=go
GOBUILD?=$(GOCMD) build
GOTEST?=$(GOCMD) test
GOCLEAN?=$(GOCMD) clean
GOTEST?=$(GOCMD) test
GOGET?=$(GOCMD) get
GOBIN?=$(GOPATH)/bin
GOMOD?=$(GOCMD) mod

OUT?=./out
IMAGE_TMP?=$(OUT)/image_temp/
IMAGE_REPO?=ghcr.io/aeraki-mesh
IMAGE_NAME?=aeraki
IMAGE_TAG?=latest
IMAGE?=$(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)
IMAGE_E2E_TAG?=`git log --format="%H" -n 1`
IMAGE_E2E?=$(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_E2E_TAG)
BINARY_NAME?=$(OUT)/aeraki
BINARY_NAME_DARWIN?=$(BINARY_NAME)-darwin
MAIN_PATH_AERAKI=./cmd/aeraki/main.go

.DEFAULT_GOAL := build

install:
	bash demo/install-aeraki.sh
install-for-tcm:
	bash demo/install-aeraki.sh mode=tcm
demo:
	bash demo/install-demo.sh default
uninstall-aeraki:
	bash demo/uninstall-aeraki.sh 
uninstall-demo:
	bash demo/uninstall-demo.sh default
demo-kafka:
	bash demo/install-demo.sh kafka
uninstall-demo-kafka:
	bash demo/uninstall-demo.sh kafka
demo-brpc:
	bash demo/install-demo.sh brpc
uninstall-demo-brpc:
	bash demo/uninstall-demo.sh brpc
test: style-check
	$(GOMOD) tidy
	$(GOTEST) -race  `go list ./... | grep -v e2e`
build: test
	CGO_ENABLED=0 GOOS=linux  $(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH_AERAKI)
build-mac: test
	CGO_ENABLED=0 GOOS=darwin  $(GOBUILD) -o $(BINARY_NAME_DARWIN) $(MAIN_PATH_AERAKI)
docker-build: build
	rm -rf $(IMAGE_TMP)
	mkdir $(IMAGE_TMP)
	cp ./docker/Dockerfile $(IMAGE_TMP)
	cp $(BINARY_NAME) $(IMAGE_TMP)
	docker build -t $(IMAGE) $(IMAGE_TMP)
	rm -rf $(IMAGE_TMP)
docker-push: docker-build
	docker push $(IMAGE)
docker-build-e2e: build
	rm -rf $(IMAGE_TMP)
	mkdir $(IMAGE_TMP)
	cp ./docker/Dockerfile $(IMAGE_TMP)
	cp $(BINARY_NAME) $(IMAGE_TMP)
	docker build -t $(IMAGE_E2E) $(IMAGE_TMP)
	rm -rf $(IMAGE_TMP)
clean:
	rm -rf $(OUT)
style-check:
	gofmt -l -d ./
	goimports -l -d ./
lint:
	golint ./...
	golangci-lint run --tests="false"
e2e-dubbo:
	go test -v github.com/aeraki-mesh/aeraki/test/e2e/dubbo/...
e2e-thrift:
	go test -v github.com/aeraki-mesh/aeraki/test/e2e/thrift/...
e2e-kafka-zookeeper:
	go test -v github.com/aeraki-mesh/aeraki/test/e2e/kafka/...
e2e-redis:
	go test -v github.com/aeraki-mesh/aeraki/test/e2e/redis/...
e2e-metaprotocol:
	go test -v github.com/aeraki-mesh/aeraki/test/e2e/metaprotocol/...
e2e: e2e-dubbo e2e-thrift e2e-kafka-zookeeper e2e-redis e2e-metaprotocol
.PHONY: build docker-build docker-push clean style-check lint e2e-dubbo e2e-thrift e2e-kafka-zookeeper e2e install demo uninstall-demo
