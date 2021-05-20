# Go parameters
GOCMD?=go
GOBUILD?=$(GOCMD) build
GOTEST?=$(GOCMD) test
GOCLEAN?=$(GOCMD) clean
GOTEST?=$(GOCMD) test
GOGET?=$(GOCMD) get
GOBIN?=$(GOPATH)/bin

# Build parameters
OUT?=./out
DOCKER_TMP?=$(OUT)/docker_temp/
DOCKER_TAG_E2E?=aeraki/aeraki:`git log --format="%H" -n 1`
DOCKER_TAG?=aeraki/aeraki:latest
BINARY_NAME?=$(OUT)/aeraki
BINARY_NAME_DARWIN?=$(BINARY_NAME)-darwin
MAIN_PATH_CONSUL_MCP=./cmd/aeraki/main.go

test: style-check
	$(GOTEST) -race  `go list ./... | grep -v e2e`
build: test
	CGO_ENABLED=0 GOOS=linux  $(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH_CONSUL_MCP)
build-mac: test
	CGO_ENABLED=0 GOOS=darwin  $(GOBUILD) -o $(BINARY_NAME_DARWIN) $(MAIN_PATH_CONSUL_MCP)
docker-build: build
	rm -rf $(DOCKER_TMP)
	mkdir $(DOCKER_TMP)
	cp ./docker/Dockerfile $(DOCKER_TMP)
	cp $(BINARY_NAME) $(DOCKER_TMP)
	docker build -t $(DOCKER_TAG) $(DOCKER_TMP)
	rm -rf $(DOCKER_TMP)
docker-push: docker-build
	docker push $(DOCKER_TAG)
docker-build-e2e: build
	rm -rf $(DOCKER_TMP)
	mkdir $(DOCKER_TMP)
	cp ./docker/Dockerfile $(DOCKER_TMP)
	cp $(BINARY_NAME) $(DOCKER_TMP)
	docker build -t $(DOCKER_TAG_E2E) $(DOCKER_TMP)
	rm -rf $(DOCKER_TMP)
clean:
	rm -rf $(OUT)
style-check:
	gofmt -l -d ./
	goimports -l -d ./
lint:
	golint ./...
	golangci-lint run --tests="false"
e2e-dubbo:
	go test -v github.com/aeraki-framework/aeraki/test/e2e/dubbo/...
e2e-thrift:
	go test -v github.com/aeraki-framework/aeraki/test/e2e/thrift/...
e2e-kafka-zookeeper:
	go test -v github.com/aeraki-framework/aeraki/test/e2e/kafka/...
e2e-redis:
	go test -v github.com/aeraki-framework/aeraki/test/e2e/redis/...
e2e: e2e-dubbo e2e-thrift e2e-kafka-zookeeper e2e-redis
.PHONY: build docker-build docker-push clean style-check lint e2e-dubbo e2e-thrift e2e-kafka-zookeeper e2e

