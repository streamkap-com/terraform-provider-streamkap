# Examples

Terraform usage examples for the Streamkap provider. They serve two purposes: manual
testing via the Terraform CLI, and (for the provider index page) documentation source.

## Layout

This repo does **not** use the default `tfplugindocs` `resource.tf` scaffold. Actual
convention:

- `provider/provider.tf` — embedded in the generated provider index page (`docs/index.md`)
  via `templates/index.md.tmpl`.
- `resources/streamkap_<name>/`
  - `basic.tf` — minimal working config.
  - `complete.tf` — full config exercising every attribute.
  - `import.sh` — the `terraform import` command.
  - transforms also carry `with_implementation.tf` (except `topic_router`, which
    routes by regex and takes no user code).
- `data-sources/streamkap_<name>/data-source.tf` — data-source config.

## Working with examples

- **Every resource page in `docs/resources/` embeds this directory.**
  `templates/resources.md.tmpl` pulls in `basic.tf` and `complete.tf` (as "Example
  Usage") and `import.sh` (as "Import"). `basic.tf` and `complete.tf` are therefore
  mandatory for every registered resource — `tfplugindocs` errors out if either is
  missing. `import.sh` is optional. Data-source pages use the stock template, which
  picks up `data-source*.tf` automatically.
- Validate locally: `make validate-examples` (runs `terraform validate` per resource dir).
- `make generate` runs `terraform fmt -recursive ./examples/` — keep files fmt-clean.
- Use placeholder hosts and `var.*` references for anything credential-shaped. These
  files are published to the Terraform Registry.
- Adding a connector via tfgen does **not** create these — author them by hand.
