IMG ?= "crossplane/sample-stack-wordpress"
VERSION ?= "0.1.0"

build:
	docker build . -t ${IMG}:${VERSION}