# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION ?= 1.24.1

AERAKI_API_VERSION ?= v1.4.1
AERAKI_API_URL ?= https://raw.githubusercontent.com/aeraki-mesh/api/${AERAKI_API_VERSION}/kubernetes/customresourcedefinitions.gen.yaml

WAIT_TIMEOUT ?= 15m

FLUENT_BIT_CHART_VERSION ?= 0.30.4
OTEL_COLLECTOR_CHART_VERSION ?= 0.60.0
TEMPO_CHART_VERSION ?= 1.3.1

##@ Kubernetes Development

YEAR := $(shell date +%Y)
CONTROLLERGEN_OBJECT_FLAGS :=  object:headerFile="$(ROOT_DIR)/tools/boilerplate/boilerplate.generatego.txt",year=$(YEAR)

.PHONY: manifests
manifests: $(tools/controller-gen) aeraki-crds ## Generate ClusterRole objects.
	@$(LOG_TARGET)
	$(tools/controller-gen) rbac:roleName=aeraki paths="./..." output:rbac:artifacts:config=charts/aeraki-helm/templates
	mv charts/aeraki-helm/templates/role.yaml charts/aeraki-helm/templates/aeraki-role.yaml

##@ Kubernetes Deployment

ifndef ignore-not-found
  ignore-not-found = true
endif

IMAGE_PULL_POLICY ?= Always

.PHONY: aeraki-crds
aeraki-crds: ## Download Aeraki CRDs from the Aeraki API repo.
	@$(LOG_TARGET)
	@mkdir -p $(OUTPUT_DIR)/
	curl -sLo $(OUTPUT_DIR)/aeraki-crds.yaml ${AERAKI_API_URL}
	mv $(OUTPUT_DIR)/aeraki-crds.yaml charts/aeraki-helm/crds/aeraki-crds.yaml

.PHONY: kube-deploy
kube-deploy: manifests ## Install Aeraki into the Kubernetes cluster specified in ~/.kube/config.
	@$(LOG_TARGET)
	helm install aeraki charts/aeraki-helm --set deployment.aeraki.imagePullPolicy=$(IMAGE_PULL_POLICY) -n istio-system --debug --timeout='$(WAIT_TIMEOUT)' --wait --wait-for-jobs

.PHONY: kube-undeploy
kube-undeploy: manifests ## Uninstall the Aeraki into the Kubernetes cluster specified in ~/.kube/config.
	@$(LOG_TARGET)
	helm uninstall aeraki -n istio-system

.PHONY: kube-demo-prepare
kube-demo-prepare:
	@$(LOG_TARGET)
	kubectl apply -f examples/kubernetes/quickstart.yaml -n default
	kubectl wait --timeout=5m -n default gateway eg --for=condition=Programmed

.PHONY: kube-demo
kube-demo: kube-demo-prepare ## Deploy a demo backend service, gatewayclass, gateway and httproute resource and test the configuration.
	@$(LOG_TARGET)
	$(eval ENVOY_SERVICE := $(shell kubectl get service -n envoy-gateway-system --selector=gateway.envoyproxy.io/owning-gateway-namespace=default,gateway.envoyproxy.io/owning-gateway-name=eg -o jsonpath='{.items[0].metadata.name}'))
	@echo -e "\nPort forward to the Envoy service using the command below"
	@echo -e "kubectl -n envoy-gateway-system port-forward service/$(ENVOY_SERVICE) 8888:80 &"
	@echo -e "\nCurl the app through Envoy proxy using the command below"
	@echo -e "curl --verbose --header \"Host: www.example.com\" http://localhost:8888/get\n"

.PHONY: kube-demo-undeploy
kube-demo-undeploy: ## Uninstall the Kubernetes resources installed from the `make kube-demo` command.
	@$(LOG_TARGET)
	kubectl delete -f examples/kubernetes/quickstart.yaml --ignore-not-found=$(ignore-not-found) -n default

.PHONY: e2e
e2e: create-cluster kube-install-image kube-deploy install-ratelimit run-e2e delete-cluster

.PHONY: install-ratelimit
install-ratelimit:
	@$(LOG_TARGET)
	kubectl apply -f examples/redis/redis.yaml
	kubectl rollout restart deployment envoy-gateway -n envoy-gateway-system
	kubectl rollout status --watch --timeout=5m -n envoy-gateway-system deployment/envoy-gateway
	kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available
	kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-ratelimit --for=condition=Available

.PHONY: run-e2e
run-e2e: prepare-e2e
	@$(LOG_TARGET)
	kubectl wait --timeout=5m -n gateway-system deployment/gateway-api-admission-server --for=condition=Available
	kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-ratelimit --for=condition=Available
	kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available
	kubectl wait --timeout=5m -n gateway-system job/gateway-api-admission --for=condition=Complete
	kubectl apply -f test/config/gatewayclass.yaml
	go test -v -tags e2e ./test/e2e --gateway-class=envoy-gateway --debug=true

.PHONY: prepare-e2e
prepare-e2e: prepare-helm-repo install-fluent-bit install-loki install-tempo install-otel-collector
	@$(LOG_TARGET)
	kubectl rollout status daemonset fluent-bit -n monitoring --timeout 5m
	kubectl rollout status statefulset loki -n monitoring --timeout 5m
	kubectl rollout status statefulset tempo -n monitoring --timeout 5m
	kubectl rollout status deployment otel-collector -n monitoring --timeout 5m

.PHONY: prepare-helm-repo
prepare-helm-repo:
	@$(LOG_TARGET)
	helm repo add fluent https://fluent.github.io/helm-charts
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
	helm repo update

.PHONY: install-fluent-bit
install-fluent-bit:
	@$(LOG_TARGET)
	helm upgrade --install fluent-bit fluent/fluent-bit -f examples/fluent-bit/helm-values.yaml -n monitoring --create-namespace --version $(FLUENT_BIT_CHART_VERSION)

.PHONY: install-loki
install-loki:
	@$(LOG_TARGET)
	kubectl apply -f examples/loki/loki.yaml -n monitoring

.PHONY: install-tempo
install-tempo:
	@$(LOG_TARGET)
	helm upgrade --install tempo grafana/tempo -f examples/tempo/helm-values.yaml -n monitoring --create-namespace --version $(TEMPO_CHART_VERSION)

.PHONY: install-otel-collector
install-otel-collector:
	@$(LOG_TARGET)
	helm upgrade --install otel-collector open-telemetry/opentelemetry-collector -f examples/otel-collector/helm-values.yaml -n monitoring --create-namespace --version $(OTEL_COLLECTOR_CHART_VERSION)

.PHONY: create-cluster
create-cluster: $(tools/kind) ## Create a kind cluster.
	@$(LOG_TARGET)
	tools/hack/create-cluster.sh

.PHONY: kube-install-image
kube-install-image: image.build $(tools/kind) ## Install the Aeraki image to a kind cluster using the provided $IMAGE and $TAG.
	@$(LOG_TARGET)
	tools/hack/kind-load-image.sh $(IMAGE) $(TAG)


.PHONY: delete-cluster
delete-cluster: $(tools/kind) ## Delete kind cluster.
	@$(LOG_TARGET)
	$(tools/kind) delete cluster --name envoy-gateway

.PHONY: generate-manifests
generate-manifests: helm-generate ## Generate Kubernetes release manifests.
	@$(LOG_TARGET)
	@$(call log, "Generating kubernetes manifests")
	mkdir -p $(OUTPUT_DIR)/
	helm template aeraki charts/aeraki-helm --include-crds --set deployment.aeraki.imagePullPolicy=$(IMAGE_PULL_POLICY) --namespace istio-system > $(OUTPUT_DIR)/install.yaml
	@$(call log, "Added: $(OUTPUT_DIR)/install.yaml")
	cp examples/kubernetes/quickstart.yaml $(OUTPUT_DIR)/quickstart.yaml
	@$(call log, "Added: $(OUTPUT_DIR)/quickstart.yaml")

.PHONY: generate-artifacts
generate-artifacts: generate-manifests generate-egctl-releases ## Generate release artifacts.
	@$(LOG_TARGET)
	cp -r $(ROOT_DIR)/release-notes/$(TAG).yaml $(OUTPUT_DIR)/release-notes.yaml
	@$(call log, "Added: $(OUTPUT_DIR)/release-notes.yaml")

.PHONY: generate-egctl-releases
generate-egctl-releases: ## Generate egctl releases
	@$(LOG_TARGET)
	mkdir -p $(OUTPUT_DIR)/
	curl -sSL https://github.com/envoyproxy/gateway/releases/download/latest/egctl_latest_darwin_amd64.tar.gz -o $(OUTPUT_DIR)/egctl_$(TAG)_darwin_amd64.tar.gz
	curl -sSL https://github.com/envoyproxy/gateway/releases/download/latest/egctl_latest_darwin_arm64.tar.gz -o $(OUTPUT_DIR)/egctl_$(TAG)_darwin_arm64.tar.gz
	curl -sSL https://github.com/envoyproxy/gateway/releases/download/latest/egctl_latest_linux_amd64.tar.gz -o $(OUTPUT_DIR)/egctl_$(TAG)_linux_amd64.tar.gz
	curl -sSL https://github.com/envoyproxy/gateway/releases/download/latest/egctl_latest_linux_arm64.tar.gz -o $(OUTPUT_DIR)/egctl_$(TAG)_linux_arm64.tar.gz

.PHONY: install-istio
install-istio:
	@$(LOG_TARGET)
	tools/hack/istio/install-istio.sh

.PHONY: uninstall-istio
uninstall-istio:
	@$(LOG_TARGET)
	tools/hack/istio/uninstall-istio.sh
