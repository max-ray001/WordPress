include package.env

build:
	docker build . -t ${PACKAGE_IMG}
.PHONY: build
