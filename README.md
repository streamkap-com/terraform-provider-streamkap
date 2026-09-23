# Streamkap Terraform Provider — v2

This branch maintains v2. The stable v3 release line lives on `main`.
v2 receives bug and security fixes through **15 October 2026**; support ends
**16 October 2026**. Published v2 releases remain available.

Before upgrading, follow the [v2 to v3 migration guide](https://github.com/streamkap-com/terraform-provider-streamkap/blob/main/docs/MIGRATION.md).

## Using the provider

Pin the v2 line until you are ready to migrate:

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "~> 2.2"
    }
  }
}

provider "streamkap" {}
```

Set `STREAMKAP_CLIENT_ID` and `STREAMKAP_SECRET` in your environment. Use the
[Registry documentation for v2.2.1](https://registry.terraform.io/providers/streamkap-com/streamkap/2.2.1/docs)
for resource schemas and examples. Adapt one example to your environment,
then run `terraform init` and review `terraform plan` before applying.

For existing configurations, retain your provider source address until you
have checked it against `terraform providers` and your state. The migration
guide covers older examples that used a different address.

## Development

Requires Terraform >= 1.0 and Go as pinned in `go.mod`.

```bash
go install .
```

For local testing, configure `~/.terraformrc` with your absolute binary path:

```hcl
provider_installation {
  dev_overrides {
    "streamkap-com/streamkap" = "/absolute/path/to/go/bin"
  }
  direct {}
}
```

Render documentation:

```bash
go generate main.go
```

Acceptance tests create and destroy real resources. Use a test tenant and the
required connector credentials:

```bash
TF_ACC=1 go test ./internal/provider -v -run '^TestAccSourcePostgreSQLResource$' -timeout 120m
```

## Releasing v2 patches

Keep `v2` and `main` histories independent. Add a matching bracketed release
heading to `CHANGELOG.md`, fetch remote refs, and run
`bash scripts/release-preflight.sh <tag>`. The tagged commit must already be on
`v2`. Branch and tag pushes require maintainer approval. Verify the published
download and Registry installation before announcing the release.
