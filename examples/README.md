# Examples

Terraform usage examples for the Streamkap provider. They serve two purposes: manual
testing via the Terraform CLI, and (for the provider index page) documentation source.

## Layout

This repo does **not** use the default `tfplugindocs` `resource.tf` / `data-source.tf`
scaffold. Actual convention:

- `provider/provider.tf` — embedded in the generated provider index page (`docs/index.md`)
  via `templates/index.md.tmpl`.
- `resources/streamkap_<name>/`
  - `basic.tf` — minimal working config.
  - `complete.tf` — full config exercising every attribute.
  - `import.sh` — the `terraform import` command.
  - transforms also carry `with_implementation.tf`.
- `data-sources/streamkap_<name>/data-source.tf` — data-source config.

## Working with examples

- Validate locally: `make validate-examples` (runs `terraform validate` per resource dir).
- `make generate` runs `terraform fmt -recursive ./examples/` — keep files fmt-clean.
- Only `provider/provider.tf` is embedded in docs today; the per-resource `basic.tf`/
  `complete.tf` are for testing and `validate-examples`.
- Adding a connector via tfgen does **not** create these — author them by hand.
