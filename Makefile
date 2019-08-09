
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

EXTENSION_PACKAGE_REGISTRY=extension-package/.registry

all: manager


# Initialize the stack bundle folder
stack-init: $(EXTENSION_PACKAGE_REGISTRY)
$(EXTENSION_PACKAGE_REGISTRY):
	mkdir -p $(EXTENSION_PACKAGE_REGISTRY)/resources
	touch $(EXTENSION_PACKAGE_REGISTRY)/app.yaml $(EXTENSION_PACKAGE_REGISTRY)/install.yaml $(EXTENSION_PACKAGE_REGISTRY)/rbac.yaml

stack-build: manifests $(EXTENSION_PACKAGE_REGISTRY)
	find $(CRD_DIR) -type f -name '*.yaml' | \
		while read filename ; do cat $$filename > \
		$(EXTENSION_PACKAGE_REGISTRY)/resources/$$( basename $${filename/.yaml/.crd.yaml} ) \
		; done

stack-install:
	kubectl apply -f config/extension/install.extension.yaml

stack-uninstall:
	kubectl delete -f config/extension/install.extension.yaml

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
docker-build: test stack-build
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

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
