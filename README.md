# Wordpress example stack

A wordpress stack with a simple controller to press wordpresses!

## Developing

### Prerequisites

This assumes that there is a crossplane running locally.
See the crossplane project for instructions on how to get that working.

Also, run a local docker registry if you don't already have one:
```
make docker-local-registry
```

### Workflow

To build and publish the stack locally, do something like:
```
make local-build
make stack-install
```

To uninstall the stack locally:

```
make stack-uninstall
```
