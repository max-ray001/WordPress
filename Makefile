include stack.env

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

clean: clean-binary
.PHONY: clean

clean-binary:
	rm -r bin
.PHONY: clean-binary

# For testing the stack end-to-end, there are a bunch of commands involved,
# so this is a convenience recipe so that the commands don't need to be run
# by hand.
integration-test:
	# The '-' prefixes ignore errors, which is what we want for the removal commands
	# in this case. The delete command will fail if the resource doesn't exist,
	# but we consider that a success.
	-kubectl delete -f config/samples/wordpress_v1alpha1_wordpressinstance.yaml
	-kubectl crossplane stack build stack-uninstall
	kubectl crossplane stack build local-build
	kubectl crossplane stack build stack-install
	# Sleeping to wait for crossplane to create the wordpress CRD so we can create
	# an instance of it. This is a bit hacky and isn't guaranteed to work all the time,
	# but it's quick to implement and works often.
	sleep 15
	kubectl apply -f config/samples/wordpress_v1alpha1_wordpressinstance.yaml
	@echo "To validate, look for the kubernetesapplicationresources created by the controller,"
	@echo "and watch their statuses. If things go well, the Service should eventually have an IP,"
	@echo "and the IP should point to a wordpress when accessed with a browser."
.PHONY: integration-test

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
	sed -i'.bak' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml
	rm ./config/default/manager_image_patch.yaml.bak

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
