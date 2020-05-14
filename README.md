# App Wordpress
Wordpress is a relatively simple sample application that only requires compute
to run its containerized binary and a MySQL database. The Wordpress application
can be deployed on top of the Crossplane Sample Stacks for GCP, AWS, and Azure
-- or any stack that provides the neccesary service catalog.

Applications allow you to define your application and its managed service
dependencies as a single installable unit.  Applications are portable in that
they create claims for infrastructure that are satisfied by different managed
service implementations depending on what stacks are installed in your
environment.

The Wordpress application can be deployed on top of stacks that provide the
default resource classes capable of satisfying the required infrastructure
claims for things like a MySQL database and a Kubernetes Cluster. All of the
following stacks provide the neccesary default resource classes in their
service catalog: [GCP Sample Stack], [AWS Sample Stack], and [Azure Sample
Stack].

You can even make your own stacks and the Wordpress application will deploy
successfully on top as long as the stack provides the neccesary default
resource classes. Checkout the [Crossplane docs] for more info.

The [templates] used in this application show how templating engines like
`helm` and `kustomize` can be used to build your own application and this repo
can be used a starting point for building your own application.

Checkout the [Wordpress Quick Start] to rapidly get started in your environment.

## Installation

Install with the following command after replacing `<version>` with the correct
one, like `0.1.0`:

```bash
kubectl crossplane package install -n default 'crossplane/app-wordpress:<version>' wordpress
```

## Usage

Here is an example CR that you can use to deploy Wordpress to a fresh new
cluster:

```yaml
apiVersion: wordpress.apps.crossplane.io/v1alpha1
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

Run `make` and then run the following command to copy the image into your
minikube node's image registry:

```bash
# Do not forget to specify <version>
docker save "crossplane/app-wordpress:<version>" | (eval "$(minikube docker-env --shell bash)" && docker load)
```

After running this, you can use the [installation](#installation) command and
the image loaded into minikube node will be picked up. 

## Release

To create and publish a release, use the upbound Jenkins jobs. You'll want to
have your release version handy.

**A note about the multibranch pipelines we use.** We're using multibranch
pipeline jobs in Jenkins, which means that Jenkins creates a new job for each
branch that it detects. This means that each time a new branch is created, such
as from the create branch job, you may need to trigger a new scan of the
repository. It also means you may need to run the job once and have it fail
before you can enter in parameters for the job.

1. First, run the [job to cut a new
   branch](https://jenkinsci.upbound.io/job/crossplaneio/job/app-wordpress/job/branch-create/),
   if needed. The branch should be named `release-MAJOR.MINOR`. For example:
   `release-0.0`.
2. If needed, edit the package version in the release branch, so that the stack
   version matches the version that will be released. Look for the stack's
   metadata block.
2. Run the [job to tag the exact
   release](https://jenkinsci.upbound.io/job/crossplaneio/job/app-wordpress/job/tag/)
   we want. For example: `v0.0.1`. If you use the one on the release branch we
   created, that's the simplest.
4. Run the [job to publish the
   release](https://jenkinsci.upbound.io/job/crossplaneio/job/app-wordpress/job/publish/).
   It's easiest if you use the branch created earlier.

[GCP Sample Stack]: https://github.com/crossplane/stack-gcp-sample
[AWS Sample Stack]: https://github.com/crossplane/stack-aws-sample
[Azure Sample Stack]: https://github.com/crossplane/stack-azure-sample
[templates]: https://github.com/crossplane/app-wordpress/tree/master/helm-chart/templates 
[Crossplane docs]: https://crossplane.github.io/docs
[Wordpress Quick Start]: docs/quickstart.md
