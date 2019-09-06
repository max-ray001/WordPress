# Wordpress example stack

A wordpress stack with a simple controller to press wordpresses!

## Using

* Install from source; see the "Developing" section for prerequisites
  - Use `kubectl crossplane stack build local-build stack-install` to build and install locally
* Create cloud provider CRDs and resource classes; see the crossplane
  wordpress workload examples
* Create a CR to represent a wordpress instance. There's a sample in
  using the sample wordpress instance in `config/samples`
* Wait for things to work; at this point, observing and debugging are
  the same as what is in the wordpress workload examples in the
  crossplane repo.

## Developing

### Prerequisites

This assumes that there is a crossplane running locally.
See the crossplane project for instructions on how to get that working.

Also, run a local docker registry if you don't already have one:
```
make docker-local-registry
```

### Workflow

To build, publish, and install the stack locally, do something like:
```
kubectl crossplane stack build local-build
kubectl crossplane stack build stack-install
```

To uninstall the stack locally:

```
kubectl crossplane stack build stack-uninstall
```

To run locally out-of-cluster:

1. Delete the deployment that the stack manager created
2. `make run`; I like to use `make manager run` to ensure that it
   rebuilds
