# Dependency-Track Terraform Provider

This is a Terraform provider for configuring an existing [Dependency-Track](https://dependencytrack.org/) server using
Terraform.

**This provider is currently intended for our internal use.** We cannot commit to supporting it for external users.
We don't currently have plans to publish it in public provider registries, although this might change in the future.
We might make changes to the provider's API without warning, including backwards-incompatible changes. You are free
to use the provider in any way permissible under the included license, but please note that we cannot commit to
supporting you with any issues that might arise.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.7
- [Go](https://golang.org/doc/install) >= 1.21

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

### Running the Acceptance tests 

In order to run the full suite of Acceptance tests, run `make testacc`.

```shell
make testacc
```

Or, to provide extra arguments to `go test`:

```shell
TESTARGS="..." make testacc
```

The Acceptance tests require a Dependency-Track API server to run against. One can either be provided, or will be
started internally by the tests in a Docker container. Providing an external server will speed up the tests, but
will make them more interdependent and may leave the server in a dirty state requiring manual cleanup.
Refer to `testutils.NewTestDependencyTrack` for how to configure the Dependency-Track API server.
