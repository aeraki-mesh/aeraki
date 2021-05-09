## most of codes here inspired by https://github.com/istio/api
## and https://github.com/istio/client-go .

SHELL := /bin/bash -ex

all: prepare gen

########################
# setup
########################

repo_dir := .
out_path := ./tmp
module_name := github.com/aeraki-framework/aeraki

protoc = protoc -I./common-protos -I.
protolock = protolock --lockdir ./crd
prototool = prototool
annotations_prep = annotations_prep
htmlproofer = htmlproofer
cue = cue-gen -paths=common-protos

go_plugin_prefix := --go_out=plugins=grpc,
go_plugin := $(go_plugin_prefix):$(out_path)

dictionaries := crd/dictionaries

########################
# prepare
########################
prepare:
	mkdir -p $(out_path)

########################
# protoc_gen_gogo*
########################

gogofast_plugin_prefix := --gogofast_out=plugins=grpc,
gogoslick_plugin_prefix := --gogoslick_out=plugins=grpc,

comma := ,
empty :=
space := $(empty) $(empty)

importmaps := \
	gogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto \
	google/protobuf/any.proto=github.com/gogo/protobuf/types \
	google/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor \
	google/protobuf/duration.proto=github.com/gogo/protobuf/types \
	google/protobuf/struct.proto=github.com/gogo/protobuf/types \
	google/protobuf/timestamp.proto=github.com/gogo/protobuf/types \
	google/protobuf/wrappers.proto=github.com/gogo/protobuf/types \
	google/rpc/error_details.proto=istio.io/gogo-genproto/googleapis/google/rpc \
	google/api/field_behavior.proto=istio.io/gogo-genproto/googleapis/google/api \

# generate mapping directive with M<proto>:<go pkg>, format for each proto file
mapping_with_spaces := $(foreach map,$(importmaps),M$(map),)
gogo_mapping := $(subst $(space),$(empty),$(mapping_with_spaces))

gogofast_plugin := $(gogofast_plugin_prefix)$(gogo_mapping):$(out_path)
gogoslick_plugin := $(gogoslick_plugin_prefix)$(gogo_mapping):$(out_path)


########################
# protoc_gen_docs
########################

protoc_gen_docs_plugin := --docs_out=warnings=true,dictionary=$(repo_dir)/$(dictionaries)/en-US,custom_word_list=$(repo_dir)/$(dictionaries)/custom.txt,mode=html_fragment_with_front_matter:$(repo_dir)/
protoc_gen_docs_plugin_per_file := --docs_out=warnings=true,dictionary=$(repo_dir)/$(dictionaries)/en-US,custom_word_list=$(repo_dir)/$(dictionaries)/custom.txt,per_file=true,mode=html_fragment_with_front_matter:$(repo_dir)/

########################
# protoc_gen_jsonshim
########################

protoc_gen_k8s_support_plugins := --jsonshim_out=$(gogo_mapping):$(out_path) --deepcopy_out=$(gogo_mapping):$(out_path)

#####################
# Generation Rules
#####################

gen: generate-redis \
     generate-dubbo \
     generate-openapi-schema \
     generate-openapi-crd \
     generate-k8s-client


#####################
# api/redis/v1alpha1/...
#####################

redis_path := api/redis/v1alpha1
redis_protos := $(wildcard $(redis_path)/*.proto)
redis_pb_gos := $(redis_protos:.proto=.pb.go)
redis_pb_docs := $(redis_protos:.proto=.pb.html)
redis_openapi := $(redis_protos:.proto=.gen.json)
redis_k8s_gos := \
	$(patsubst $(redis_path)/%.proto,$(redis_path)/%_json.gen.go,$(shell grep -l "^ *oneof " $(redis_protos))) \
	$(patsubst $(redis_path)/%.proto,$(redis_path)/%_deepcopy.gen.go,$(shell grep -l "+kubetype-gen" $(redis_protos)))

$(redis_pb_gos) $(redis_pb_docs) $(redis_k8s_gos): $(redis_protos)
	@$(protolock) status
	@$(protoc) $(gogofast_plugin) $(protoc_gen_k8s_support_plugins) $(protoc_gen_docs_plugin)$(redis_path) $^
	@cp -r $(out_path)/$(module_name)/api/redis/v1alpha1/* api/redis/v1alpha1 && rm -rf $(out_path)/$(module_name)/api/redis/v1alpha1

generate-redis: $(redis_pb_gos) $(redis_pb_docs)

clean-redis:
	@rm -fr $(redis_pb_gos) $(redis_pb_docs)

#####################
# api/dubbo/v1alpha1/...
#####################

dubbo_path := api/dubbo/v1alpha1
dubbo_protos := $(wildcard $(dubbo_path)/*.proto)
dubbo_pb_gos := $(dubbo_protos:.proto=.pb.go)
dubbo_pb_docs := $(dubbo_protos:.proto=.pb.html)
dubbo_openapi := $(dubbo_protos:.proto=.gen.json)
dubbo_k8s_gos := \
	$(patsubst $(dubbo_path)/%.proto,$(dubbo_path)/%_json.gen.go,$(shell grep -l "^ *oneof " $(dubbo_protos))) \
	$(patsubst $(dubbo_path)/%.proto,$(dubbo_path)/%_deepcopy.gen.go,$(shell grep -l "+kubetype-gen" $(dubbo_protos)))

$(dubbo_pb_gos) $(dubbo_pb_docs) $(dubbo_k8s_gos): $(dubbo_protos)
	@$(protolock) status
	@$(protoc) $(gogofast_plugin) $(protoc_gen_k8s_support_plugins) $(protoc_gen_docs_plugin)$(dubbo_path) $^
	@cp -r $(out_path)/$(module_name)/api/dubbo/v1alpha1/* api/dubbo/v1alpha1 && rm -rf $(out_path)/$(module_name)/api/dubbo/v1alpha1

generate-dubbo: $(dubbo_pb_gos) $(dubbo_pb_docs)

clean-dubbo:
	@rm -fr $(dubbo_pb_gos) $(dubbo_pb_docs)


#####################
# OpenAPI Schema
#####################

all_protos := \
	$(redis_protos) $(dubbo_protos)

all_openapi := \
	$(redis_openapi) $(dubbo_openapi)

all_openapi_crd :=./crd/kubernetes/customresourcedefinitions.gen.yaml

$(all_openapi): $(all_protos)
	@$(cue) -f=$(repo_dir)/cue.yaml

# The fields are added at the end to generate snake cases. This is a temporary solution to accommodate some wrong namings that currently exist.
$(all_openapi_crd): $(all_protos)
	@$(cue) -f=$(repo_dir)/cue.yaml --crd=true -snake=jwksUri,apiKeys,apiSpecs,includedPaths,jwtHeaders,triggerRules,excludedPaths,mirrorPercent
ifeq ($(VERIFY_CRDS_SCHEMA),1)
	@$(validate_crds) check_equal_schema --kinds RedisService,RedisDestination,DubboAuthorizationPolicy --versions
	v1alpha1 --file $
	(all_openapi_crd)
endif


generate-openapi-schema: $(all_openapi)

generate-openapi-crd: $(all_openapi_crd)


########################
# kubernetes code generators
########################
kubetype_gen = kubetype-gen
deepcopy_gen = deepcopy-gen
client_gen = client-gen
lister_gen = lister-gen
informer_gen = informer-gen

empty:=
space := $(empty) $(empty)
comma := ,

# source packages to scan for kubetype-gen tags
kube_source_packages = $(subst $(space),$(empty), \
	$(module_name)/api/redis/v1alpha1 \
	$(space), $(module_name)/api/dubbo/v1alpha1 \
	)

# base output package for generated files
kube_base_output_package = $(module_name)/client-go/pkg
# base output package for kubernetes types, register, etc...
kube_api_base_package = $(kube_base_output_package)/apis
# source packages to scan for kubernetes generator tags, e.g. deepcopy-gen, client-gen, etc.
# these should correspond to the output packages from kubetype-gen
kube_api_packages = $(subst $(space),$(empty), \
	$(kube_api_base_package)/redis/v1alpha1 \
	$(space), $(kube_api_base_package)/dubbo/v1alpha1 \
	)
# base output package used by kubernetes client-gen
kube_clientset_package = $(kube_base_output_package)/clientset
# clientset name used by kubernetes client-gen
kube_clientset_name = versioned
# base output package used by kubernetes lister-gen
kube_listers_package = $(kube_base_output_package)/listers
# base output package used by kubernetes informer-gen
kube_informers_package = $(kube_base_output_package)/informers

# file header text
kube_go_header_text = ./crd/header.go.txt

ifeq ($(IN_BUILD_CONTAINER),1)
    output_base=
	# k8s code generators rely on GOPATH, using $GOPATH/src as the base package
	# directory.  Using --output-base . does not work, as that ends up generating
	# code into ./<package>, e.g. ./aeraki.io/client-go/pkg/apis/...  To work
	# around this, we'll just let k8s generate the code where it wants and copy
	# back to where it should have been generated.
	move_generated=cp -r $(GOPATH)/src/$(kube_base_output_package)/ . && rm -rf $(GOPATH)/src/$(kube_base_output_package)/
else
	# nothing special for local builds
	output_base=-o $(out_path)
	move_generated=cp -r $(out_path)/$(kube_base_output_package)/ ./client-go/pkg/ &&  rm -rf ./tmp/$(kube_base_output_package)/
endif

rename_generated_files=\
	find $(subst $(module_name)/, $(empty), $(subst $(comma), $(space), $(kube_api_packages)) $(kube_clientset_package) $(kube_listers_package) $(kube_informers_package)) \
	-name '*.go' -and -not -name 'doc.go' -and -not -name '*.gen.go' -type f -exec sh -c 'mv "$$1" "$${1%.go}".gen.go' - '{}' \;

.PHONY: generate-k8s-client
generate-k8s-client:
	# generate kube api type wrappers for istio types
	@$(kubetype_gen) --input-dirs $(kube_source_packages) $(output_base) --output-package $(kube_api_base_package)
	@$(move_generated)
	# generate deepcopy for kube api types
	@$(deepcopy_gen) --input-dirs $(kube_api_packages) $(output_base) -O zz_generated.deepcopy  -h $(kube_go_header_text)
	# generate clientsets for kube api types
	@$(client_gen) --clientset-name $(kube_clientset_name) $(output_base) --input-base "" --input  $(kube_api_packages) --output-package $(kube_clientset_package) -h $(kube_go_header_text)
	# generate listers for kube api types
	@$(lister_gen) --input-dirs $(kube_api_packages) $(output_base) --output-package $(kube_listers_package) -h $(kube_go_header_text)
	# generate informers for kube api types
	@$(informer_gen) --input-dirs $(kube_api_packages) $(output_base) --versioned-clientset-package $(kube_clientset_package)/$(kube_clientset_name) --listers-package $(kube_listers_package) --output-package $(kube_informers_package) -h $(kube_go_header_text)
	@$(move_generated)
	@$(rename_generated_files)
