# Wordpress Sample Stack

This is a Crossplane Stack that you can use to deploy Wordpress into a
`KubernetesCluster` using a `MySQLInstance` database in the cloud.

## Installation

Install with the following command after replacing `<version>` with the
correct one, like `0.1.0`:

```bash
kubectl crossplane stack install -n default 'crossplane/sample-stack-wordpress:<version>' wordpress
```

## Usage

Here is an example CR that you can use to deploy Wordpress to a fresh
new cluster:

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

Run `make` and then run the following command to copy the image into
your minikube node's image registry:

```bash
# Do not forget to specify <version>
docker save "crossplane/sample-stack-wordpress:<version>" | (eval "$(minikube docker-env --shell bash)" && docker load)
```

After running this, you can use the [installation](#installation)
command and the image loaded into minikube node will be picked up. 

## Release

To create and publish a release, use the upbound Jenkins jobs. You'll
want to have your release version handy.

**A note about the multibranch pipelines we use.** We're using
multibranch pipeline jobs in Jenkins, which means that Jenkins creates a
new job for each branch that it detects. This means that each time a new
branch is created, such as from the create branch job, you may need to
trigger a new scan of the repository. It also means you may need to run
the job once and have it fail before you can enter in parameters for the
job.

1. First, run the [job to cut a new branch](https://jenkinsci.upbound.io/job/crossplaneio/job/sample-stack-wordpress/job/branch-create/),
   if needed. The branch should be named `release-MAJOR.MINOR`. For
   example: `release-0.0`.
2. If needed, edit the stack version in the release branch, so that the
   stack version matches the version that will be released. Look for the
   stack's metadata block.
2. Run the [job to tag the exact release](https://jenkinsci.upbound.io/job/crossplaneio/job/sample-stack-wordpress/job/tag/)
   we want. For example: `v0.0.1`. If you use the one on the release
   branch we created, that's the simplest.
4. Run the [job to publish the release](https://jenkinsci.upbound.io/job/crossplaneio/job/sample-stack-wordpress/job/publish/).
   It's easiest if you use the branch created earlier.
