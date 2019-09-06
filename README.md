# Wordpress example stack

A wordpress stack with a simple controller to press wordpresses!

## Using

### Install the Crossplane stack CLI

First, install the [Crossplane stack
CLI](https://github.com/crossplaneio/crossplane-cli#installation).

### Install

If `kubectl` is set up to talk to a Crossplane control cluster, the
stack can be installed using the stack cli:

```
kubectl crossplane stack install crossplane/sample-stack-wordpress
```

### Create wordpresses

Before wordpresses will provision, the Crossplane control cluster must
be configured to connect to a provider.

Once a provider is configured, starting the process of creating a
wordpress is easy. Create a wordpress instance [like the sample
shows](./config/samples/wordpress_v1alpha1_wordpressinstance.yaml):

```
apiVersion: wordpress.samples.stacks.crossplane.io/v1alpha1
kind: WordpressInstance
metadata:
  name: wordpressinstance-sample
```

The stack (and Crossplane) will take care of the rest.

## Developing

### Prerequisites

This assumes that there is a crossplane running locally.
See the crossplane project for instructions on how to get that working.

It also assumes that you have the [crossplane
cli](https://github.com/crossplaneio/crossplane-cli) installed.

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
