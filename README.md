# Sample Stack Wordpress

This is an example of a Crossplane Stack which wraps Wordpress. The
Stack uses the Template Stacks approach, which means it does not include
a controller of its own. Instead, it uses a Crossplane-provided
controller; the Wordpress Stack gives the provided controller a
configuration to define its behavior.

## Developing

To build:

```sh
make
```

## Releasing

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
