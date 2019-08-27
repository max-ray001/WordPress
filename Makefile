
# Image URL to use all building/pushing image targets
IMG ?= crossplane-examples/wordpress-stack:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

GO111MODULE ?= on
export GO111MODULE

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

CRD_DIR=config/crd/bases
LOCAL_OVERRIDES_DIR=config/samples/overrides

STACK_PACKAGE=stack-package
STACK_PACKAGE_REGISTRY=$(STACK_PACKAGE)/.registry
STACK_PACKAGE_REGISTRY_SOURCE=config/stack

all: manager

################################################
#
# Below this until marked otherwise is where
# most of the build customizations beyond
# what kubebuilder generates live.
#
# Here are some other customizations of note:
# - Set IMG above to be more specific
# - Add GO111MODULE=on
# - Add COPY of stack bundle in Dockerfile
# - Add some other make variables
# - Made docker-build recipe work well on MacOS
#
################################################

clean: clean-stack-package clean-binary
.PHONY: clean

clean-stack-package:
	rm -r $(STACK_PACKAGE)
.PHONY: clean-stack-package

clean-binary:
	rm -r bin
.PHONY: clean-binary

# Initialize the stack bundle folder
$(STACK_PACKAGE_REGISTRY):
	mkdir -p $(STACK_PACKAGE_REGISTRY)/resources
	touch $(STACK_PACKAGE_REGISTRY)/app.yaml $(STACK_PACKAGE_REGISTRY)/install.yaml $(STACK_PACKAGE_REGISTRY)/rbac.yaml

stack-build: manager manifests $(STACK_PACKAGE_REGISTRY)
	# Copy CRDs over
	#
	# The reason this looks complicated is because it is
	# preserving the original crd filenames and changing
	# *.yaml to *.crd.yaml.
	#
	# An alternate and simpler-looking approach would
	# be to cat all of the files into a single crd.yaml.
	find $(CRD_DIR) -type f -name '*.yaml' | \
		while read filename ; do cat $$filename > \
		$(STACK_PACKAGE_REGISTRY)/resources/$$( basename $${filename/.yaml/.crd.yaml} ) \
		; done
	cp -r $(STACK_PACKAGE_REGISTRY_SOURCE)/* $(STACK_PACKAGE_REGISTRY)
.PHONY: stack-build

# A local docker registry can be used to publish and consume images locally, which
# is convenient during development, as it simulates the whole lifecycle of the
# Stack, end-to-end.
docker-local-registry:
	docker run -d -p 5000:5000 --restart=always --name registry registry:2
.PHONY: docker-local-registry

# Tagging the image with the address of the local registry is a necessary step of
# publishing the image to the local registry.
docker-local-tag:
	docker tag ${IMG} localhost:5000/${IMG}
.PHONY: docker-local-tag

# When we are developing locally, this target will publish our container image
# to the local registry.
docker-local-push: docker-local-tag
	docker push localhost:5000/${IMG}
.PHONY: docker-local-push

# Sooo ideally this wouldn't be a single line, but the idea here is that when we're
# developing locally, we want to use our locally-published container image for the
# Stack's controller container. The container image is specified in the install
# yaml for the Stack. This means we need two versions of the install:
#
# * One for "regular", production publishing
# * One for development
#
# The way this has been implemented here is by having the base install yaml be the
# production install, with a yaml patch on the side for use during development.
# *This* make recipe creates the development install yaml, which requires doing
# some patching of the original install and putting it back in the stack package
# directory. It's done here as a post-processing step after the stack package
# has been generated, which is why the output to a copy and then rename step is
# needed. This is not the only way to implement this functionality.
#
# The implementation here is general, in the sense that any other yamls in the
# overrides directory will be patched into their corresponding files in the
# stack package. It assumes that all of the yamls are only one level deep.
stack-local-build: stack-build
	find $(LOCAL_OVERRIDES_DIR) -maxdepth 1 -type f -name '*.yaml' | \
		while read filename ; do \
			kubectl patch --dry-run --filename $(STACK_PACKAGE_REGISTRY)/$$( basename $$filename ) \
				--type strategic --output yaml --patch "$$( cat $$filename )" > $(STACK_PACKAGE_REGISTRY)/$$( basename $$filename ).new && \
			mv $(STACK_PACKAGE_REGISTRY)/$$( basename $$filename ).new $(STACK_PACKAGE_REGISTRY)/$$( basename $$filename ) \
		; done
.PHONY: stack-local-build

# Convenience for building a local bundle by running the steps in the right order and with a single command
local-build: stack-local-build docker-build docker-local-push
.PHONY: local-build

# Install a locally-built stack using the sample stack installation CR
stack-install:
	kubectl apply -f config/samples/install.stack.yaml
.PHONY: stack-install

stack-uninstall:
	kubectl delete -f config/samples/install.stack.yaml
.PHONY: stack-uninstall

######################################
#
# Below here, the recipes are (mostly)
# from the kubebuilder boilerplate,
# with the exception of the edits
# noted in the docker-build recipe.
#
######################################

# Run tests
test: generate fmt vet manifests
	go test ./api/... ./controllers/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crd/bases

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crd/bases
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=$(CRD_DIR)

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	@# The argument to sed -i and the subsequent rm make the in-place sed work well on MacOS.
	@# There is no good way to do an in-place replacement with sed without leaving behind a
	@# temporary file.
	sed -i '.bak' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml
	rm ./config/default/manager_image_patch.yaml.bak

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.4
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
