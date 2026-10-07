# AGENTS.md — Provider development rules

Rules for agents changing the Streamkap Terraform provider (`streamkap-com/streamkap`).
Writing Terraform configs *with* the provider is covered by `docs/USAGE.md`.

## Versions and branches

- `main` = v3.x stable, default target for all work. `v2` = v2.x legacy, bug and security fixes only through 15 October 2026.
- Histories are independent: never merge `main` and `v2`; a shared fix is one commit per line. Pull and confirm the branch before starting.
- Pushing a tag publishes a public release. Surface the exact `git tag`/`git push` commands and stop for approval; never push tags or branches unprompted. Procedure: `docs/RELEASING.md`.

## Documentation map

| Before touching… | Read |
|---|---|
| Terraform config, resource catalog, patterns, import, error recovery | `docs/USAGE.md` |
| CRUD flow, marshaling, API client, backend contract quirks, CI workflows, test tiers | `docs/ARCHITECTURE.md` |
| tfgen, `overrides.json`, deprecated aliases, adding a connector, post-regen checklist | `docs/CODE_GENERATOR.md` |
| Releases, tags, version pins, stable promotion, security gate | `docs/RELEASING.md` |
| v2 → v3 upgrade guide | `templates/guides/migration.md` (source); `go generate main.go` renders `docs/guides/migration.md`; `docs/MIGRATION.md` is the GitHub entry point |
| Registry pages (`docs/index.md`, `docs/resources/`, `docs/data-sources/`) | Generated — never hand-edit; `make generate` |
| User-visible change | `CHANGELOG.md`, before tagging |

Keep docs in sync in the same PR: new/removed resource → `docs/USAGE.md` catalog + `docs/ARCHITECTURE.md` counts + `make generate`; deprecated alias → `TestAcc<Connector>_MigrationFromLegacy` in `internal/provider/migration_test.go` + `templates/guides/migration.md`; tfgen mapping change → `docs/CODE_GENERATOR.md`; CRUD/retry/pagination/quirk change → `docs/ARCHITECTURE.md` + the quirks list below. If a doc is wrong, fix it — never write a parallel one. Registry-published guides belong in `docs/guides/`.

`docs/audits/` and `docs/plans/` are gitignored: local scratch, invisible to clones. Do not commit them or edit `.gitignore` to change that. `env.md` is gitignored and holds live credentials: never commit, quote or link it.

## Public repository — content hygiene

Public repo. Never commit internal ticket IDs/URLs, customer or tenant identifiers, real credentials, internal hostnames, verbatim production traces, or unannounced roadmap details. Public GitHub issue numbers (`#75`) are fine. Reference surfaces as `https://api.streamkap.com` and `https://docs.streamkap.com`. Flag any new occurrences you find.

## Authentication

- OAuth2 client credentials from `STREAMKAP_CLIENT_ID` / `STREAMKAP_SECRET` (optional `STREAMKAP_HOST`, default `https://api.streamkap.com`). Examples and docs use env vars and `sensitive = true` variables, never inline literals.
- Tests auto-load `.env` via godotenv; a `TF_ACC=1` line there turns any unfiltered `go test` into a live-API run. Use the `make` targets — `make test` clears `TF_ACC`. Multiline Snowflake PEM keys: `source scripts/load-pem-keys.sh`.
- Local provider override: `dev_overrides` in `~/.terraformrc` mapping `streamkap-com/streamkap` to `$GOPATH/bin` (where `make install` lands the binary) plus `direct {}`.

## Commands

`make help` lists all targets.

| Command | Use |
|---|---|
| `make test-all` | Unit + schema-compat + validators, no API (same as `make test`) |
| `make testacc` | Acceptance, `TF_ACC=1`, ~15m, real API |
| `make test-migration` | v2 → v3 migration acceptance, ~30m |
| `make snapshots` | Rewrite schema-compat snapshots after an intentional schema change; read the diff |
| `make sweep` | Delete orphaned test resources |
| `make lint` | golangci-lint |
| `STREAMKAP_BACKEND_PATH=<backend clone> make generate` | Regenerate schemas (tfgen) then registry docs (tfplugindocs) |
| `go run ./cmd/tfgen generate --backend-path=<path> --entity-type=sources --connector=postgresql` | One connector |
| `bash scripts/release-preflight.sh <tag>` | Tag/branch/changelog gate before tagging |

Single acceptance test: `TF_ACC=1 go test -v ./internal/provider -run '^TestAccSourcePostgreSQLResource$'`.

## Code generation

- Regenerate **only** with `make generate`. `go generate ./...` renders docs before tfgen runs, so a new field lands in `internal/generated/*.go` but is missing from `docs/resources/*.md`.
- `scripts/codegen-preflight.sh` aborts when `STREAMKAP_BACKEND_PATH` is unset/unreadable or the backend is not on `main`. `ALLOW_NONMAIN=1` only when told to target a branch; restore the backend's original branch afterwards.
- `internal/generated/` is rewritten on every regen. Fix bugs in `cmd/tfgen/{parser,generator}.go`, `cmd/tfgen/overrides.json` or the backend `configuration.latest.json`, regenerate, commit source + output together, and grep every connector for the same defect class in the same PR.
- `internal/resource/{source,destination,transform}/<connector>_generated.go` are **hand-maintained** despite the suffix: wrapper structs, CRUD wiring, deprecated aliases. Edit freely.
- Unmarked credential field in the backend → extend `isSecretField` (`cmd/tfgen/generator.go`); `overrides.json` fields need their own `Sensitive`.
- A new resource needs `examples/resources/streamkap_<name>/{basic,complete}.tf` (tfplugindocs fails without both), a `TestSchemaBackwardsCompatibility_*` case, and a snapshot (`TestEveryResourceHasSchemaSnapshot`).
- After a regen work the post-regen checklist in `docs/CODE_GENERATOR.md`: snapshot diff is the changelog; a `changed=` `Sensitive` flag is a security regression until proven otherwise; `removed=` can orphan a deprecated alias (`TestDeprecatedAliasTargetsExist` never reads the backend spec).

## Deprecated attribute pattern (v2 → v3 aliases)

Backend field kept, Terraform name changed → add an alias in the wrapper, not in tfgen: a struct embedding the generated model, an `Optional+Computed` attribute with `DeprecationMessage` + `ConflictsWith(<new name>)`, and a `fieldMappings` row to the same API field. Example: `internal/resource/source/postgresql_generated.go`. Then add `TestAcc<Connector>_MigrationFromLegacy` and a migration-guide entry. Not aliasable (document in the guide + exceptions map): required fields, renames to a different backend field, type changes. Details: `docs/CODE_GENERATOR.md#deprecated-attribute-aliases`.

## API quirks (non-obvious)

Rationale and provider code paths for each: `docs/ARCHITECTURE.md#backend-contract-quirks`.

- POST/PUT/DELETE on `/sources`, `/destinations`, `/pipelines` need `&wait=false` (mock URLs too). Sources Read needs `?secret_returned=true`.
- Lists default `page_size=10` (max 100); paginate until `resp.Total` or tenants with >10 resources are truncated. Name filter is `partial_name` only; match exact name client-side.
- 422 "already exists" on create: **no connector resource adopts** — fail with recovery guidance (`terraform import` vs. deposed `create_before_destroy` instance). Only tags adopt.
- `periodic_audit` must be round-tripped on every pipeline update (`preservePeriodicAudit`); omitting the key `$unset`s a UI-configured audit.
- Empty map `{}` clears a mapping, null leaves it; keep nil and empty Go maps distinct.
- Unknown `admin_tenant_id` / `admin_service_id` in Configure → diagnostics before auth; unknown-as-empty targets the default tenant.
- `Optional+Computed` fields echo null when the backend gates them off; refill from plan (Create/Update) or prior state (Read) via `shared.DefaultedAttrNames` + `FillNullFields`. The echo mismatch (`was cty.StringVal(""), but now null`) hits defaults, deprecated aliases and `<...>` placeholders — fix the whole class across every connector, not one field.
- Secrets echo null even with `secret_returned=true`. Create/Update restore every planned `Sensitive` string (`shared.CaptureFields`/`PreserveKnownFields`); Read restores from state only where the echo is null, so a rotated credential still shows.
- Kafka ACL topic field: send `topic_name`, responses return `name`; `api.KafkaACL` unmarshals both. Any Pydantic `Field(alias=...)` response needs the same check. Kafka usernames: 3-24 chars, `[a-zA-Z0-9-]`, no leading/trailing hyphen, no underscores. Kafka users have no GET; Read filters the list.
- Client credential `secret` is real only in the create response; `service_id` is ForceNew; `role_ids` is a Set.
- Transform `deploy = true`: terminal failure is an error, but keep the saved ID in state. Redact implementation bodies and validation input values before logging.
- `streamkap_topics.messages_7d/30d` are always null (deprecated).
- Use `UseStateForUnknown()` for stable computed fields; omit it for derived fields so updates can recompute them.

## Testing

Tiers and what each verifies: `docs/ARCHITECTURE.md#testing-architecture`.

- Schema-compat fails on *any* drift from a snapshot (added/removed attribute, flipped `Required`/`Optional`/`Computed`/`Sensitive`, nested fields included). Intentional change → `make snapshots`, review the diff, commit it.
- Migration tests start from v2.2.0, require surviving IDs and a converged plan. A skipped case is not upgrade evidence; inspect changed defaults even for in-place updates.
- Offline API tests use `httpmock` (`internal/api/client_test.go`, `internal/provider/state_conflict_test.go`). Recorded fixtures redact bodies as well as headers.
- Acceptance env: `TF_ACC=1`, `STREAMKAP_CLIENT_ID`, `STREAMKAP_SECRET`; optional `STREAMKAP_HOST`, `UPDATE_SNAPSHOTS`, `TF_LOG`.
- Handed a reported issue (GitHub, Slack, customer error): confirm it is real and not already fixed by pending work, report that verdict, then change code.

## Conventions

- Every resource/data source attribute has `Description` and `MarkdownDescription`; hand-added attributes document enums ("Valid values: …"), defaults ("Defaults to …") and a `**Security:**` note when sensitive. tfgen emits these automatically.
- Use `streamkap-com/streamkap` in Terraform configs and dev overrides; the Go module path is `github.com/streamkap-com/terraform-provider-streamkap`.
- Connector status values (read-only): `Active`, `Paused`, `Stopped`, `Broken`, `Starting`, `Unassigned`, `Unknown`.
- **Never log a connector `Config`/`configMap`** — keyed by API field name, holds decrypted credentials. Request bodies go through `redactSensitiveJSON` (`internal/api/redact.go`); anything logged outside `internal/api` bypasses it. Secrets travel under dotted Kafka-Connect names (`api.key`, `snowflake.private.key`), so new redaction patterns treat `.` as a separator.
- Bump version pins (`examples/provider/provider.tf`, `README.md`, `templates/guides/migration.md`) in the same commit as the CHANGELOG section, before tagging.
