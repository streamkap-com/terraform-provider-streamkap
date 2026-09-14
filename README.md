# Terraform Provider

## v2 maintenance

When v3 becomes stable, v2 becomes the legacy maintenance line. Bug fixes and
security patches continue through **15 October 2026**; new features and
connectors are v3-only. From **16 October 2026**, v2 receives no further fixes
or support. Published v2 releases remain available. Pin `version = "~> 2.2"`
until you are ready to upgrade.

## Releasing v2 patches

Use `main` before branch promotion and `v2` afterwards. Prepare a new bracketed
changelog heading such as `## [2.2.1] - <date>`, preserving older release entries.
Fetch remote refs and run `bash scripts/release-preflight.sh <tag>`. The tagged
commit must already be on the release branch. Branch and tag pushes require
maintainer approval; verify the published download and registry installation
before announcing a release.

## High Level Design

The Streamkap Terraform provider is a wrapper over the Streamkap API implemented in the [backend project](../backend/)
Resources are streamkap sources, destinations, pipelines and transforms.

* [postgresql source](./internal/provider/source_postgresql_resource_test.go) -> [postgresql source details from backend](../backend/app/sources/plugins/postgresql/configuration.latest.json)
* [kafkadirect source] TODO -> [kafkadirect source details from backend](../backend/app/sources/plugins/kafkadirect/configuration.latest.jsons)
* [databricks destination](./internal/provider/destination_databricks_resource_test.go) -> [databricks destination details from backend](../backend/app/destinations/plugins/databricks/configuration.latest.json)
* [kafka destination] TODO -> [kafka destination details from backend](../backend/app/destinations/plugins/kafka/configuration.latest.json)


## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.27.1 (building the provider)

## Using the provider

Fill this in for each provider

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (
see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin`
directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

### Testing with terraform

Configure `~/.terraformrc`, replace `$GOBIN_PATH` with your `$GOPATH/bin`
```hcl
provider_installation {
  dev_overrides {
    "github.com/streamkap-com/streamkap" = "$GOBIN_PATH"
  }
  direct {}
}
```

Install provider with
```shell
go install .
``````

Write your module, can see the example in [examples/full](/examples/full/)