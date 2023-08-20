# This is a wrapper to manage helm chart
#
# All make targets related to helmß are defined in this file.

include tools/make/env.mk

OCI_REGISTRY ?= oci://docker.io/aeraki
CHART_NAME ?= aeraki-helm
CHART_VERSION ?= ${RELEASE_VERSION}

##@ Helm
helm-package:
helm-package: ## Package aeraki helm chart.
helm-package: helm-generate
	@$(LOG_TARGET)
	helm package charts/${CHART_NAME} --app-version ${TAG} --version ${CHART_VERSION} --destination ${OUTPUT_DIR}/charts/

helm-push:
helm-push: ## Push aeraki helm chart to OCI registry.
	@$(LOG_TARGET)
	helm push ${OUTPUT_DIR}/charts/${CHART_NAME}-${CHART_VERSION}.tgz ${OCI_REGISTRY}

helm-install:
helm-install: helm-generate ## Install aeraki helm chart from OCI registry.
	@$(LOG_TARGET)
	helm install eg ${OCI_REGISTRY}/${CHART_NAME} --version ${CHART_VERSION} -n istio-system

helm-generate:
	ImageRepository=${IMAGE} ImageTag=${TAG} envsubst < charts/aeraki-helm/values.tmpl.yaml > ./charts/aeraki-helm/values.yaml
