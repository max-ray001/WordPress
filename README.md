# Wordpress Sample Stack

This is a Crossplane Stack that you can use to deploy Wordpress into a `KubernetesCluster` using a `MySQLInstance` database in the cloud. Here is an example CR:

## Installation

Install with the following command after replacing `<version>` with the correct one, like `0.1.0`:
```bash
kubectl crossplane stack install -n default 'crossplane/sample-stack-wordpress:<version>' wordpress
```

## Usage

Here is an example CR that you can use to deploy Wordpress to a fresh new cluster:

```yaml
apiVersion: wordpress.samples.stacks.crossplane.io/v1alpha1
kind: WordpressInstance
metadata:
  name: testme
spec:
# You can use UseExistingTarget as well to schedule to a KubernetesTarget in the
# same namespace randomly.
  provisionPolicy: ProvisionNewCluster

#  This is the default value.
# image: wordpress:4.6.1-apache
```

## Build

Run `make`.

## Test Locally

### Minikube

Run `make` and then run the following command to copy the image into your minikube node's image registry:

```bash
# Do not forget to specify <version>
docker save "crossplane/sample-stack-wordpress:<version>" | (eval "$(minikube docker-env --shell bash)" && docker load)
```

After running this, you can use the [installation](#installation) command and the image loaded into minikube node will be picked up. 