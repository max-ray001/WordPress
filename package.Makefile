include package.env

build:
	docker build . -t ${PACKAGE_IMG}
.PHONY: build

publish:
	docker push ${PACKAGE_IMG}
.PHONY: publish
