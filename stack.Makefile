include stack.env

build:
	docker build . -t ${STACK_IMG}
.PHONY: build

publish:
	docker push ${STACK_IMG}
.PHONY: publish
