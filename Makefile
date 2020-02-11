IMG ?= "crossplane/sample-stack-wordpress"
VERSION ?= "0.0.2"

build:
	docker build . -t ${IMG}:${VERSION}