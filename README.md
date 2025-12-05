# Terraform Provider

## High Level Design

The Streamkap Terraform provider is a wrapper over the Streamkap API implemented in the [backend project](../backend/)
Resources are streamkap sources, destinations, pipelines and transforms.

* [postgresql source](./internal/provider/source_postgresql_resource_test.go) -> [postgresql source details from backend](../backend/app/sources/plugins/postgresql/configuration.latest.json)
* [kafkadirect source] TODO -> [kafkadirect source details from backend](../backend/app/sources/plugins/kafkadirect/configuration.latest.jsons)
* [databricks destination](./internal/provider/destination_databricks_resource_test.go) -> [databricks destination details from backend](../backend/app/destinations/plugins/databricks/configuration.latest.json)
* [kafka destination] TODO -> [kafka destination details from backend](../backend/app/destinations/plugins/kafka/configuration.latest.json)


## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.20

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