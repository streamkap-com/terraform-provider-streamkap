# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed (breaking, v3 beta)
- `streamkap_topic_metrics` now decodes the backend's topic-keyed table metrics.
  `results` gains `id`, `partition_count`, `replication_factor`, `retention_ms`,
  `last_message_timestamp`, `snapshot_status_json` and `record_error_total`.
  **`messages_in`, `messages_out`, `bytes_in`, `bytes_out`, `lag` and
  `avg_latency_ms` are now always null** — this endpoint has never returned them,
  so the previous schema reported nulls under names the API does not emit. They
  remain in the schema, deprecated, so existing configurations still parse.
  The `time_interval` and `time_unit` inputs are likewise deprecated and ignored.
  Outputs, monitoring or expressions consuming the six legacy metrics must move
  to the new attributes or to another metrics endpoint before upgrading.
  Topics the tenant does not own are omitted from `results` and now raise a
  warning naming them.

### Known issues
- Upgrading a resource created under v2 can plan an in-place `update` with no
  configuration change, because v3 adds optional attributes that carry
  client-side defaults. Confirmed on the Kafka Direct source and the Databricks
  destination; see `docs/MIGRATION.md` → "Expect an in-place update on your
  first v3 plan". No replacement occurs and no data is lost.

### Fixed
- `streamkap_client_credential` failed every create and update with
  "Value Conversion Error ... Path: roles": the computed `roles` list is unknown
  in the plan and was decoded into a Go slice. It is now a framework list type.
- Correct topic-metrics decoding and expose the broker metadata and status the
  API returns. Preserve legacy inputs and result fields with deprecation notices;
  unavailable metrics and ambiguous entity associations are null.
- Cap and redact API error bodies surfaced in diagnostics.
- Deprecate `messages_7d` and `messages_30d` on `streamkap_topics`; the topic
  details API has never returned them, so both are always null.
- Stop deleting a pipeline's periodic row-level audit. The backend removes
  `periodic_audit` whenever an update omits it, so every apply that touched a
  pipeline silently discarded an audit configured outside Terraform. The
  provider now reads and carries it through, narrowing its topics (with a
  warning) if the pipeline no longer streams them.
- Reject `admin_service_id` without `admin_tenant_id`. The backend ignores the
  service header unless the tenant header is present, so the request ran against
  the credential's own tenant instead of the intended one.
- Say when a request was replayed after a transient failure, so an
  "already exists" error from a retried create is not read as a name collision.
- Preserve redacted error context when the API returns null or empty details.
- Honor cancellation during authentication and reject unknown admin scope IDs
  before configuring an API client.
- Avoid retrying failed client-credential creation requests, which can issue
  duplicate credentials when a response is lost.
- Report terminal transform deployment failures as errors and support read timeouts.
- Preserve explicitly empty connector maps and tag descriptions.
- Keep topic read failures visible unless the API confirms a missing topic.
- Refresh backend-derived MongoDB hostnames after connection-string updates
  while preserving existing attribute configurability.
- Correct required attributes, numeric ports and full signal-table paths in examples.
- Use sensitive variables consistently for example credentials.
- Replace fixed beta pins in embedded examples with guidance to select the release
  matching the documentation.

### Security
- Update gRPC to v1.83.1 to address CVE-2026-84304.
- Redact transform implementation payloads and structured validation inputs from
  API diagnostics and logs.
- Update Go, dependencies and pinned workflow actions. Gate releases on security
  scans and serialize acceptance jobs that share staging fixtures.

### Changed
- Validate migration from the current stable v2.2.0 baseline.
- Track nested attributes and block fields in schema snapshots; reject unsupported
  backend controls during generation.
- Mark beta GitHub releases as prereleases automatically and check the exact
  changelog heading before publishing.
- Consolidate development documentation and correct migration guidance.

## [3.0.0-beta.30] - 2026-08-25 (Pre-release)

### Added
- **`streamkap_kafka_user`, `streamkap_client_credential` and the
  `streamkap_roles` data source are registered again.** They were unregistered
  in beta.4 with the source left in the tree, so every reference to them failed
  with "provider does not support resource type" while `README.md`,
  `AGENTS.md`, `CHANGELOG.md` and `docs/MIGRATION.md` still advertised them.
  Each now ships with examples, registry documentation, a schema snapshot,
  acceptance tests and a sweeper.
- **`streamkap_client_credential` supports in-place updates.** `role_ids` and
  `description` now go through `PATCH /auth/client-credentials/{client_id}`
  instead of forcing replacement. Replacing a credential issues a new secret and
  invalidates the old one, so a role change used to silently break whatever was
  authenticating with it. `service_id` still forces replacement — the update
  endpoint does not accept it.

### Fixed
- **`streamkap_kafka_user` could never be applied.** The API accepts an ACL's
  topic under `topic_name` but returns it under `name`, so the decoded value was
  always empty and every apply failed with "Provider produced inconsistent
  result after apply" on `kafka_acls[*].topic_name`. The client now writes the
  documented alias and reads both spellings.
- **`streamkap_kafka_user` accepted usernames the API rejects.** The validator
  allowed 1-2 character names, which the backend refuses with a 400. Minimum
  length is now 3, matching the backend.
- **`streamkap_client_credential` could fail on apply for multi-role
  credentials.** `role_ids` was an ordered list, but the API resolves roles in
  its own catalog order; a mismatch surfaced as an inconsistent-result error.
  It is now a set.
- **`streamkap_client_credential.roles` and `data.streamkap_roles.roles` were
  declared as configuration blocks** rather than computed attributes, so
  server-populated values conflicted with the empty block list in the plan.

### Changed
- **`streamkap_client_credential.role_ids` is now a set, not a list.** Existing
  configurations keep working; ordering no longer produces a diff.

## [3.0.0-beta.29] - 2026-08-20 (Pre-release)

### Added
- **`streamkap_destination_snowflake`: `transforms_key_to_value_fields_include_list`**
  — comma-separated list of key fields to copy into each record's value so they
  land as columns in Snowflake. `*` copies every key field; `<topic>.<field>`
  patterns scope by topic and field name (e.g. `*.pk,*.sk`). Leave unset to
  disable. Snowflake-only: the backend ships the KeyToValue SMT in the Snowflake
  plugin config rather than the shared destination config.

### Security
- Bumped the indirect `golang.org/x/mod` to `v0.40.0` (pulling `x/crypto`,
  `x/net`, `x/text` and `x/tools` up with it), clearing CVE-2026-56864 and
  CVE-2026-56865. Both concern module fetching via a malicious GOSUMDB/GOPROXY;
  the package is build tooling and is not linked into the provider binary, so
  released builds were never exposed.

## [3.0.0-beta.28] - 2026-08-18 (Pre-release)

### Added
- **`streamkap_destination_iceberg`**: `iceberg_catalog_oauth2_server_uri`,
  `iceberg_catalog_audience`, and `iceberg_catalog_resource` — OAuth2 token
  endpoint, audience, and resource parameters for REST catalogs whose
  authorization server is not the catalog endpoint itself.
- **`streamkap_source_dynamodb`**: `incremental_snapshot_interval_ms` (default
  `6000`) — delay between snapshot chunks, to throttle snapshot throughput and
  keep the destination from backing up.
- **`streamkap_source_kafkadirect`**: `records_carry_streamkap_metadata`
  (default `false`) — set when the topic is fed by another Streamkap pipeline
  and records already carry `_streamkap_offset`, `_streamkap_ts_ms` and
  `_streamkap_source_ts_ms`, so the destination skips its own InsertField and
  upstream offsets survive.

### Fixed
- **`data.streamkap_topic` failed on every read.** The client decoded
  `kafka.partitions` as an integer, but the API returns an object
  (`{count, replication_factor, ...}`), so the response failed to unmarshal and
  the data source returned an error instead of topic details. `kafka.configs`
  was read under dotted Kafka property names (`retention.ms`,
  `cleanup.policy`); the API sends `retention_ms` and `cleanup_policy` as
  strings. `retention_ms` is parsed back to an integer, so the attribute type is
  unchanged.
- **`make generate` failed outside a `terraform-provider-streamkap` directory.**
  `tfplugindocs` derives the provider name from the checkout directory name, so
  a differently-named clone produced `data source entitled "<dir>" does not
  exist` after deleting `docs/`. The name is now passed explicitly, and the
  docs-drift workflow runs `main.go`'s own directives so the two cannot diverge.

### Changed
- **`streamkap_source_dynamodb`: `incremental_snapshot_chunk_size` default is
  now `8192`** (was `32768`), matching the backend. Configurations that omit the
  attribute will show a diff on the next plan.

## [3.0.0-beta.27] - 2026-08-05 (Pre-release)

### Added
- **`streamkap_destination_s3`: `format_output_envelope`** (bool, default
  `true`) — controls whether each output record is wrapped in an envelope
  with Kafka metadata (key, offset, timestamp, headers) alongside the value.
  Set to `false` to write only the record's own value structure. No effect on
  CSV; for Parquet, only applies when the value is a record or map.
- **`streamkap_destination_s3`: cross-account IAM role authentication** —
  `aws_auth_mode` (`Access Keys` default, or `Cross-Account Role`),
  `aws_sts_role_arn`, and `aws_sts_role_external_id` let Streamkap assume an
  IAM role in your account instead of using long-lived access keys. This
  schema catch-up reflects backend support already live
  that had not yet been regenerated into the provider. `aws_access_key_id`
  and `aws_secret_access_key` change from `Required` to `Optional`/`Computed`
  to accommodate the new mode; existing configurations using access keys are
  unaffected.
- **Mask Field transform options on every destination** —
  `transforms_mask_field_fields_include_list`,
  `transforms_mask_field_fields_exclude_list`,
  `transforms_mask_field_mask_function` (`SHA256_TRUNCATE` default,
  `MD5_TRUNCATE`, `SHA256`, `MD5`, `REDACT`, `FIXED`, `NULLIFY`),
  `transforms_mask_field_mask_salt`, `transforms_mask_field_mask_char`,
  `transforms_mask_field_mask_fixed_value`, and
  `transforms_mask_field_replace_null_with_default` mask (anonymise) string
  columns in-place on the way into the destination.
- **`streamkap_destination_s3`**: `file_max_records` and
  `aws_s3_part_size_bytes`.
- **`streamkap_source_mysql` / `streamkap_source_mariadb`**:
  `inconsistent_schema_handling_mode` plus the streaming-snapshot tuning set
  `streamkap_snapshot_chunk_size_bytes`,
  `streamkap_snapshot_max_split_size_bytes`, `streamkap_snapshot_parallelism`,
  `streamkap_snapshot_state_refresh_ms`.
- **`streamkap_source_sqlserver`**: `streamkap_snapshot_max_split_size_bytes`.
- **`streamkap_source_oracle` / `streamkap_source_oracleaws`**: `lob_enabled`,
  `log_mining_strategy`, `post_processors_reselect_enabled`,
  `reselector_reselect_error_handling_mode`.
- **`streamkap_source_postgresql` / `streamkap_source_alloydb` /
  `streamkap_source_supabase`**: `post_processors_reselect_enabled` and
  `reselector_reselect_error_handling_mode`.
- **`streamkap_source_planetscale`**: `database_ssl_disabled`,
  `vtgate_mysql_port`, `vitess_set_basic_authentication_header`, and the
  streaming-snapshot tuning set above.

### Security
- **`google.golang.org/grpc` bumped to v1.82.1**, clearing GHSA-hrxh-6v49-42gf
  (xDS RBAC and HTTP/2 vulnerabilities), the sole HIGH finding in the
  dependency scan.
- **`streamkap_destination_iceberg`: `iceberg_catalog_s3_access_key_id` is now
  marked sensitive.** The backend ships it without `encrypt`; tfgen's forced
  `Sensitive` rule for `*access_key_id` (introduced in beta.26) now applies to
  it on regeneration. A Terraform `output` exposing it needs
  `sensitive = true`.

### Breaking
- **`streamkap_destination_s3`: `file_name_prefix` is removed** — the backend
  dropped the field. Use `file_name_template` to control file naming.
  (`streamkap_destination_gcs`/`_r2`/`_starburst` keep their
  `file_name_prefix`.)
- **`streamkap_source_postgresql` / `streamkap_source_alloydb` /
  `streamkap_source_supabase`: `post_processors` is removed.** The backend now
  derives it from the new `post_processors_reselect_enabled` toggle; set that
  instead of the raw processor list.

## [3.0.0-beta.26] - 2026-07-14 (Pre-release)

Remediation of the 2026-07-11 provider audit. Grouped by what a user actually notices.

### Security
- **`aws_access_key_id` is now marked sensitive** on `streamkap_destination_s3`,
  `streamkap_destination_starburst` and `streamkap_destination_r2`. The backend
  ships `encrypt: true` for it on the DynamoDB source but not on these three, so
  the same credential was masked on one connector and printed in plan output on
  the others. tfgen now forces `Sensitive` for `*access_key_id`,
  `*secret_access_key` and `*secret_key`, so the fix survives regeneration. If
  you expose one of these through a Terraform `output`, that output now needs
  `sensitive = true`.
- **CI now scans for secrets** (gitleaks), runs on every PR, and blocks a commit
  that would introduce one.

### Breaking
- **`streamkap_source_sqlserver`: `snapshot_custom_table_config` is removed.**
  It mapped to a backend field (`streamkap.snapshot.custom.table.config.user.defined`)
  that **does not exist**, so every value ever set was accepted by Terraform and
  silently discarded — the per-table chunk counts never reached the connector.
  Remove it from your configuration; use `snapshot_parallelism` (and
  `streamkap_snapshot_chunk_size_bytes`) to tune snapshot throughput. See
  `docs/MIGRATION.md`.
- **Sources and destinations no longer adopt an existing record on "already
  exists".** Create now fails with recovery guidance instead. Adopting is
  indistinguishable, from inside the client, from a `create_before_destroy`
  replace whose deposed instance still holds the name — and in that case the new
  state entry inherits the deposed entry's backend id, so Terraform's next step
  deletes the resource it just created. Pipelines and transforms already refused
  to adopt for this reason (it destroyed a customer's pipelines). Tags still
  adopt deliberately. Recovery for a genuinely lost create response is
  `terraform import`, which the error message spells out.
- **`file_name_template` no longer carries a client-side default** on
  `streamkap_destination_s3`, `_gcs`, `_r2`, `_azblob` and `_starburst`. The
  sinks declare a default of `{{topic}}-{{partition}}-{{start_offset}}` that they
  do not store: the connector appends the extension it derives from the output
  format and compression. The value is now whatever the backend computes, and
  shows as `(known after apply)` when an input it derives from changes.

### Fixed
- **`streamkap_destination_s3` could never be created from scratch.** Every
  `terraform apply` failed with `Provider produced inconsistent result after
  apply: .file_name_template: was cty.StringVal("…{{start_offset}}"), but now
  cty.StringVal("…{{start_offset}}.json.gz")`, because the provider planned a
  default the sink rewrites. It went unnoticed because Create used to adopt an
  existing record on a name collision and the tests reused one long-lived
  destination whose stored value already matched. See the Breaking note above.
- **A transient gateway error during a read no longer kills the apply.** Reads
  went out with no retry at all, so a single `502 Bad Gateway` on a GET failed
  `terraform apply`/`import` outright. GETs are idempotent and are now replayed.
- **Long applies no longer die on an expired token.** The OAuth token was
  fetched once at provider startup and never refreshed, so any apply outliving
  the token TTL failed every remaining resource with an opaque 401. The client
  now renews ahead of expiry and retries once on a 401.
- **`terraform destroy` no longer fails on a resource deleted out of band.**
  Delete is now idempotent on 404 for sources, destinations, pipelines,
  transforms, tags, kafka users and client credentials; previously it errored and
  left the operator to run `terraform state rm`.
- **No more perpetual diff on credentials the backend echoes as null.** Refresh
  nulled such a secret in state while the config still held it, producing a diff
  on every plan and a spurious update on every apply. Read now restores the
  prior state value, but only where the API returned null — a credential rotated
  outside Terraform still shows up as drift.
- **`streamkap_topics` returned at most 10 topics.** It sent `limit`/`offset`,
  which the backend does not accept; it now paginates properly. Its `entity_ids`
  filter also collapsed to a single arbitrary ID, because the backend expects one
  comma-separated value rather than a repeated parameter.
- **Retries now key off the HTTP status**, not substring-matching the message. A
  429 whose text omitted the digits was never retried, while a 400 mentioning
  e.g. "port 4290" was retried for minutes.
- `streamkap_pipeline`'s `destination` now validates its required fields at plan
  time instead of failing at apply, matching `source`.
- Connector `Read` is now covered by the `timeouts` block, so a hung backend
  can't block refresh indefinitely.
- Corrected `examples/`: the S3 source used a nonexistent `aws_s3_object_prefix`,
  and the pipeline example set two Snowflake attributes that no longer exist.

### Added
- **Every resource page on the Terraform Registry now shows an Example Usage
  section** (previously none of the 59 did — the examples existed but were named
  in a way `tfplugindocs` did not pick up).
- Missing examples for `streamkap_transform_topic_router`, the Salesforce and
  Zendesk webhook sources, `streamkap_destination_pinecone`, `streamkap_tag`, and
  the `streamkap_tags` data source.
- Multi-select attributes (e.g. `format_output_fields`) now validate their
  allowed values instead of accepting any string.
- **Retry behaviour is configurable** via `api.Config.Retry` (defaults to 5
  attempts, 10–60s backoff). Useful for a tenant hitting sustained 429s, or an
  environment that would rather fail fast.

### Changed
- **Deprecated v2 attributes now have a stated end date: they are removed in
  v4.0, not at v3.0.0 stable.** `docs/MIGRATION.md` said all three things in
  three different places. The aliases exist so a v2 configuration can reach v3
  without a rewrite, so removing them the moment v3 goes stable would defeat
  their purpose. They keep working, with a deprecation warning, for all of v3.x.

### Changed (contributors)
- **Acceptance runs sweep the tenant before and after.** A leaked fixture holds
  database-level resources — a signal table, a replication slot — keyed to the
  database rather than the connector's name, so one leftover failed every later
  run with "Signal table … is already in use by another connector". Fixed fixture
  names plus adopt-on-exists used to absorb this; with adoption gone and names
  unique per run, the leaks have to be swept instead.
- Five connectors are quarantined from the PR gate — ClickHouse, Oracle,
  OracleAWS and PlanetScale point at unreachable or stale test databases, and
  SQL Server's signal table is held by a connector created outside Terraform.
  Each is listed with its cause in `scripts/acceptance-tests.txt` and still runs
  under `make testacc`.
- **The VCR/cassette test tier is removed.** It was scaffolding: the single
  `TestIntegration_` function began with an unconditional `t.Skip` placed
  *before* the `UPDATE_CASSETTES` check, so it never ran and `make cassettes`
  could never record. No cassette ever existed. It was counted as coverage by
  two separate audits. Offline API coverage is httpmock
  (`internal/api/client_test.go`, `internal/provider/state_conflict_test.go`);
  add there.
- **CI actually gates PRs now**: build, vet, lint, unit, schema-compat and
  validator tests. None of this ran before, so the schema-snapshot drift guard —
  which the project relies on to catch a lost `Sensitive` flag — was never
  executed by CI. Releases are gated on the same checks plus a changelog entry.
- **tfgen fails loudly instead of emitting wrong code.** A malformed connector
  config, an unreadable `configurations_for_all.json`, two attributes colliding
  on one Terraform name, or an override pointing at a backend field that no
  longer exists each used to warn-and-continue, producing a silently incomplete
  schema. The `snapshot_custom_table_config` removal above is the first bug this
  caught.
- Schema snapshots now record each attribute's **type**, so a String→Int64 change
  is caught; data sources are snapshotted too.
- Migration tests now actually set the deprecated v2 aliases they exist to
  protect; acceptance fixtures no longer collide between concurrent runs.
- The pre-commit hook no longer runs `go generate ./...`, which regenerated docs
  against a stale schema and, with `STREAMKAP_BACKEND_PATH` unset, emitted wrong
  output.
- `golangci-lint` upgraded to v2; the Go 1.24 analysis pin is gone.

## [3.0.0-beta.25] - 2026-07-10 (Pre-release)

### Security
- **`api_key` is now marked sensitive on every webhook source.**
  `streamkap_source_webhook`, `streamkap_source_shopify_webhook` and
  `streamkap_source_stripe_webhook` exposed the key in plan output;
  `streamkap_source_salesforce_webhook` and `streamkap_source_zendesk_webhook`
  had been hand-patched but lost the fix on every regeneration. The backend
  declares `api.key` with neither `encrypt: true` nor `control: "password"`, so
  tfgen now forces `Sensitive` for `api_key`/`*_api_key` and the fix survives
  codegen. See `docs/MIGRATION.md` if you reference `api_key` in an output.
- **`http_headers_authorization` is now marked sensitive on
  `streamkap_destination_httpsink`.** It carries the literal `Authorization`
  header value (e.g. `Bearer <token>`) and was exposed the same way.
- **Connector and transform Create/Update no longer log the resolved config map.**
  `TF_LOG=DEBUG terraform apply` printed every credential — database passwords,
  Snowflake private keys, webhook API keys — in plaintext for every connector,
  because the map was formatted with `%+v` and never passed through redaction.
  The redacted request body logged by the API client already covers this.
- **Debug-log redaction now matches the dotted wire field names.** Connector
  configs are sent with Kafka-Connect names (`api.key`,
  `snowflake.private.key`), but the redaction pattern only allowed `_` and `-`
  as separators, so those two keys were logged in the clear by the one code
  path built to prevent exactly that.
- **Dependency CVEs.** `golang.org/x/net` `v0.49.0` → `v0.56.0` and
  `golang.org/x/crypto` `v0.48.0` → `v0.54.0`. `x/net/http2` and `x/net/idna`
  are linked into the released provider through gRPC, exposing CVE-2026-33814
  (HTTP/2 denial of service) and CVE-2026-39821 (privilege escalation via
  Punycode label processing).

### Added
- **`streamkap_destination_s3`**: `aws_auth_mode` (`Access Keys` or
  `Cross-Account Role`), `aws_sts_role_arn` and `aws_sts_role_external_id` for
  cross-account IAM role assumption. `aws_access_key_id` and
  `aws_secret_access_key` are no longer required — they are only needed for the
  `Access Keys` mode.
- **`streamkap_source_postgresql`**: `streamkap_snapshot_max_split_size_bytes`,
  splitting oversized tables into disjoint ctid page ranges during snapshot.

### Removed
- **`streamkap_source_sqlserver`**: the deprecated `snapshot_large_table_threshold`
  alias. Its replacement, `streamkap_snapshot_large_table_threshold`, was dropped
  from the schema when the backend removed the underlying config field, so the
  alias pointed at an attribute that no longer existed and errored at plan time
  for anyone who set it. See `docs/MIGRATION.md` for the full list of attributes
  the backend removed.

### Fixed
- `make test` (and therefore `make test-all`) no longer runs the acceptance suite
  against the live API. It is the only no-API target without a `-run` filter, and
  `-short` does not gate `resource.Test`, so a developer `.env` containing
  `TF_ACC=1` turned it into a ~10-minute production run.
- Schema compatibility snapshots were stale: `tags` was missing from 48 of them
  because the commit that added the attribute never ran `make snapshots`. The
  compat test now fails on any snapshot drift instead of logging additions as
  informational, so a baseline cannot silently rot again.

## [3.0.0-beta.24] - 2026-07-02 (Pre-release)

### Changed
- **`streamkap_destination_bigquery`** now provisions the Aiven BigQuery sink
  (Storage Write API, append-only) — the only supported BigQuery destination.
  The schema is replaced to match:
  - **Removed:** `bigquery_json`, `table_name_prefix`, `bigquery_region`,
    `custom_bigquery_partition_field`, `custom_bigquery_cluster_field`,
    `bigquery_time_based_partition`.
  - **Added:** `keyfile` (required, sensitive — service-account JSON),
    `default_dataset` (required), `time_partitioning_type` (default `DAY`,
    accepts `NONE` for non-partitioned tables), `custom_partition_field`,
    `custom_clustering_fields`, `custom_partition_expiration_days` (optional —
    auto-drop partitions older than N days), `auto_create_tables`
    (default `true`), `allow_new_big_query_fields` (default `true`),
    `allow_big_query_required_field_relaxation` (default `true`).

  This is a breaking change for the beta line; update existing
  `streamkap_destination_bigquery` configurations to the new attributes.

## [3.0.0-beta.23] - 2026-06-26 (Pre-release)

### Fixed
- **"Provider produced inconsistent result after apply: inconsistent values for
  sensitive attribute"** on connector Create/Update. The Streamkap backend does
  not faithfully echo every secret back even with `secret_returned=true` — it
  returns `null` for a secret whose stored value is absent or decrypts to
  `"null"` (e.g. Snowflake `snowflake_private_key_passphrase` when the key is not
  passphrase-secured). The provider used the API echo as the new state, so when
  the planned value was known but the echo differed, Terraform aborted the apply.
  Create/Update now restore the user-supplied value for every `Sensitive` string
  attribute after applying the API response. Applies to all source and
  destination connectors.

## [3.0.0-beta.22] - 2026-06-22 (Pre-release)

### Added
- **PostgreSQL: `heartbeat_use_logical_message`** (bool, default `false`) — run
  `SELECT pg_logical_emit_message(true, ...)` on each beat to advance the
  replication slot. Works on PG14+ primaries with a SELECT-only role and is
  compatible with read-only mode; no `streamkap_heartbeat` table or write
  grant required on the source.

### Changed
- **Kafka-only heartbeat mode is now reachable across all source connectors
  that support heartbeats.** Setting `heartbeat_enabled = true` while leaving
  `heartbeat_data_collection_schema_or_database` blank emits heartbeats to a
  Kafka topic only — keeping the poll loop active and offsets advancing on
  low-traffic sources without requiring a `streamkap_heartbeat` table or a
  write grant in the source database. Setting the schema/database field still
  enables source-table heartbeat mode. Affects: PostgreSQL, MySQL, MariaDB,
  Oracle, AlloyDB, Supabase, OracleAWS, SqlServerAWS.
- **SqlServerAWS: `heartbeat_data_collection_schema_or_database` no longer
  defaults to `"streamkap"`.** Behavior parity with the other source
  connectors — the field is now Optional with no default, so Kafka-only
  heartbeat mode is reachable by leaving it blank. Existing configurations
  that explicitly set this attribute are unaffected.
- Heartbeat attribute descriptions on the affected sources were rewritten to
  document Kafka-only vs source-table modes and the conditions under which
  each path applies.
- **Three new source connectors:** `streamkap_source_informix` (IBM Informix
  CDC), `streamkap_source_shopify_webhook`, and `streamkap_source_stripe_webhook`.
  Each ships with `basic`/`complete` examples and an import recipe. Schemas were
  generated from the backend `configuration.latest.json` plugin specs on `main`
  and cross-checked field-by-field against those specs.

### Changed
- **Regenerated connector schemas against the current production backend.** Beyond
  description/help-text refreshes across most database sources, the substantive
  changes are: new optional attributes `post_processors` (postgresql, alloydb,
  supabase), `heartbeat_use_logical_message` (postgresql),
  `streamkap_snapshot_chunk_size_bytes` and `streamkap_snapshot_state_refresh_ms`
  (postgresql, sqlserveraws), and `table_name_prefix` (destination clickhouse).
  The backend dropped `streamkap_snapshot_large_table_threshold` (postgresql,
  sqlserveraws) and `streamkap_snapshot_custom_table_config` (postgresql);
  configurations setting these must remove them.

### Fixed
- **`streamkap_topics` and `streamkap_topic` no longer fail to read.** The
  backend changed the `serialization` field on `/topics` responses from a string
  to an object, so listing topics failed with `cannot unmarshal object into Go
  struct field TopicDetails.result.serialization of type string`. The data
  sources now model `serialization` as a nested block exposing `key_format`,
  `value_format`, `key_converter`, `value_converter`, and
  `schema_registry_enabled`. Configurations that referenced `serialization` as a
  string must switch to `serialization.value_format`.

## [3.0.0-beta.21] - 2026-06-15 (Pre-release)

### Fixed
- **`streamkap_source_kafkadirect` and `streamkap_destination_kafkadirect` no
  longer expose configuration attributes the backend rejects.** tfgen merged
  `configurations_for_all.json` (the shared source/destination config) into every
  connector, but the backend's `_load_global_configuration()` returns `{}` for
  `kafkadirect` — it resolves against its plugin config alone. As a result both
  resources advertised ~30 phantom attributes (`quote_identifiers`,
  `preserve_null_values`, every `transforms_*`, `consumer_override_max_poll_records`,
  the source's `insert_topic_name_enabled` / `transforms_value_to_key_*`, etc.)
  that are not valid Kafka Direct config. tfgen now skips the common-config merge
  for `kafkadirect`, matching the backend. The destination keeps `password` and
  `whitelist_ips`; the source keeps `topic_prefix`, `topic_include_list`,
  `format`, and `schemas_enable` (plus the `kafka_format` deprecated alias).
  Configurations that set any of the removed attributes must drop them.

## [3.0.0-beta.20] - 2026-06-09 (Pre-release)

### Fixed
- **Numeric defaults stored as strings in the backend schema are now emitted
  correctly instead of `0`.** tfgen read `number`/`slider`-control defaults via
  a getter that only handled JSON numbers, so fields whose backend default is a
  string (e.g. `"180000"`) fell through to `0`. With `Optional + Computed +
  Default`, omitting such a field made Terraform send `0` to the API, overriding
  the backend's intended default. Affected attributes and their corrected
  defaults: `streamkap_source_dynamodb` `poll_timeout_ms` (`0` → `180000`),
  `signal_kafka_poll_timeout_ms` (`0` → `1000`), `incremental_snapshot_chunk_size`
  (`0` → `32768`), `incremental_snapshot_max_threads` (`0` → `8`),
  `full_export_expiration_time_ms` (`0` → `86400000`); and
  `streamkap_destination_databricks` `connection_timeout` (`0` → `180`).
  Resources that left these unset will show a one-time plan diff aligning them
  to the correct default.

## [3.0.0-beta.19] - 2026-06-03 (Pre-release)

### Fixed
- **`streamkap_destination_postgresql` docs now list the `TopicRegexRouter`
  attributes.** The schema fields shipped in beta.18, but their registry
  documentation was missing because `go generate ./...` ran `tfplugindocs`
  (root `main.go`) before `tfgen` (`internal/generated`), rendering docs against
  the pre-regen schema. The `make generate` target now runs `tfgen` first, then
  `tfplugindocs`, so docs always match the freshly generated schemas.

## [3.0.0-beta.18] - 2026-06-03 (Pre-release)

### Added
- **`streamkap_destination_postgresql` now exposes the `TopicRegexRouter`
  transforms.** Four optional attributes let you rename destination tables by
  applying up to two regex-based rewrite rules to the incoming topic name:
  `transforms_topic_regex_router1_regex` / `transforms_topic_regex_router1_replacement`
  and `transforms_topic_regex_router2_regex` / `transforms_topic_regex_router2_replacement`.
  Both replacement fields default to `$0` (the full match); leave a regex empty
  to skip that rule.

### Fixed
- **Regenerated provider docs to match the committed schemas.** Several
  `docs/resources/*.md` pages had drifted ahead of the generated Go schemas
  (showing `post_processors`, `heartbeat_use_logical_message`,
  ClickHouse `table_name_prefix`, and richer heartbeat/SSH descriptions that
  only exist on backend feature branches, not production). The docs now reflect
  the schemas the provider actually ships. No schema change.

## [3.0.0-beta.17] - 2026-05-21 (Pre-release)

### Added
- **`streamkap_pipeline` now supports `topic_auto_discovery_transforms`.** This
  optional list of `{ transform_id, regex }` rules lets a pipeline automatically
  pick up a transform's output topics whose names match a regex, without
  enumerating them in `transforms[].topics`. Use it when a transform produces
  topics whose names are generated dynamically (e.g. a topic-router / fan-out
  transform) and are therefore not known when the pipeline is created. Topic
  resolution happens server-side; the list is stored and returned unchanged.

### Fixed
- **Map config attributes are restored to their correct `map` type** (regression
  introduced in earlier v3 betas). A code-generation bug (`tfgen` resolved
  `overrides.json` relative to the working directory, so `go generate ./...`
  loaded zero overrides) caused three map attributes to be emitted as plain
  strings — and `snapshot_custom_table_config` to be renamed to
  `streamkap_snapshot_custom_table_config`:
    - `streamkap_destination_snowflake.auto_qa_dedupe_table_mapping`
    - `streamkap_destination_clickhouse.topics_config_map`
    - `streamkap_source_sqlserver.snapshot_custom_table_config`
  These now match v2.1.x (and the intended v3 design): map attributes under their
  original names. **If you set any of these as a string on a v3 beta ≤ beta.16,
  revert to the map form** (see `docs/MIGRATION.md`). Migrating from v2.1.x
  requires no change — these were always maps there. The override path is now
  resolved cwd-independently so a full regen can't silently drop it again.
- **`connector_status` no longer triggers "Provider produced inconsistent
  result after apply" when updating a source or destination.** Updates are sent
  with `wait=false`, so the API response reports the transient desired-state
  `Pending Update` while the plan still carries the prior known status
  (e.g. `Active`). The provider was overwriting state with that transient value,
  which Terraform rejected as plan/apply divergence. The Update path now keeps
  the planned status (it is volatile and refreshed by Read; the backend
  reconciler settles the connector back to `Active` within seconds).
- **Boolean connector config fields no longer drift on every apply.** The
  backend returns connector config as string-encoded values, so booleans arrive
  as `"true"` / `"false"`. `helper.GetTfCfgBool` only accepted a native Go
  `bool` and returned null for the string form, so any bool field the backend
  backfills with a default (e.g. PostgreSQL destination
  `transforms_mark_columns_as_required_fields_include_all`,
  `transforms_oversized_records_replace_null_with_default`,
  `transforms_to_decimal_j_truncate_to_max_precision`,
  `transforms_to_jsonb_j_convert_all_json`) read back null and showed a
  perpetual diff. `GetTfCfgBool` now coerces string-encoded booleans, mirroring
  `GetTfCfgInt64` / `GetTfCfgFloat64`. Fixes the drift for every bool attribute
  on every connector.
- **`streamkap_pipeline` `source.topics` is hardened against the issue #78 bug
  class.** `api2Model` now strips a leading `source_<id>.` prefix from returned
  source topics (only that exact prefix — never a naive split, so dotted names
  like `default.MyTable` survive), so state stays stable across plan/apply even
  if the API echoes the raw `topic_id` form or state was written by an older
  beta. Note for DynamoDB sources: send source topics in the source's catalog
  form (e.g. `default.<table>`) so they register as selected in the pipeline.

## [3.0.0-beta.16] - 2026-05-11 (Pre-release)

### Changed
- **`streamkap_pipeline` and `streamkap_transform_*` no longer auto-adopt on
  duplicate-name conflicts (409 / 422 "already exists").** The previous
  adopt-on-conflict path was unsafe under
  `lifecycle { create_before_destroy = true }`: Terraform creates the new
  instance first while the deposed instance still occupies the
  `{tenant_id, service_id, name}` slot, the provider would adopt the
  deposed's backend record (returning the same backend id from Create), and
  Terraform's subsequent destroy of the deposed entry would DELETE the same
  backend record the live state had just been pointed at — silently
  destroying the customer's live pipeline / transform. A customer hit this on
  pipelines and had deposed transforms in the same state that would have
  triggered the same issue once the pipeline path was unblocked
  (trace in `0_Terraform Apply.txt`). The provider now refuses to auto-adopt
  on these two resources and surfaces a clear error with two recovery paths:
    1. Remove `create_before_destroy = true` from the resource's
       `lifecycle` (Streamkap enforces unique pipeline / transform names per
       tenant + service, so that lifecycle does not work).
    2. `terraform import streamkap_pipeline.<name> <pipeline_id>` or
       `terraform import streamkap_transform_<type>.<name> <transform_id>`
       when recovering from a previous apply whose response was lost.
  The original adopt-on-conflict still applies to `streamkap_source_*`,
  `streamkap_destination_*`, and `streamkap_tag`. The **same data-loss path
  exists in principle** for those resources — adopt-on-conflict under
  `create_before_destroy` is unsafe for any backend that enforces
  name-uniqueness, regardless of resource type. We are holding their adopt
  path back from this PR because (a) the reported customer wasn't hitting
  it on those resources, (b) existing acceptance tests rely on adopt-on-422
  behavior there, and (c) tightening them would break legitimate
  timed-out-create recovery workflows. This is **not** a statement that
  they are safe; track removing their adopt path in a follow-up before any
  customer relies on `lifecycle { create_before_destroy = true }` against
  those resources.
- **`streamkap_transform_*` — `deploy = true` with `replay_window = "0"`
  failed with "Replay window must be a valid duration like 3d, 10m, 2h, 30s".**
  The provider's schema documents `"0"` as "continue from last position"
  (semantically identical to leaving the field unset → backend's
  `start_time = None` → use the latest offset). But the backend's preview-
  deploy parser uses a strict `(\d+)([smhd])` regex
  (`python-be-streamkap/app/utils/api/v2/api_transforms_utils.py:352`) that
  rejects a bare `"0"`. The deploy client now treats `"0"` and `""` as
  equivalent and omits the query parameter in both cases, restoring the
  documented behavior without requiring a backend change.
- **Non-JSON error responses surface a useful message.** When an upstream
  layer (nginx, load balancer, ingress) returned HTML or another non-JSON
  body for a 5xx, the client previously bubbled up a bare
  `invalid character '<' looking for beginning of value` JSON-decode error
  — the HTTP status and body content were silently dropped, which dead-ended
  customer debugging. Error responses that fail to decode as the standard
  `{"detail": "..."}` shape now surface the actual status code and a
  truncated body snippet so the real failure is visible.

## [3.0.0-beta.15] - 2026-05-08 (Pre-release)

### Added
- **Tags on every entity.** `streamkap_source_*`, `streamkap_destination_*`,
  `streamkap_transform_*`, and `streamkap_topic` now expose a `tags = [...]`
  attribute (Set of tag IDs) — at parity with `streamkap_pipeline`, which
  already had it. Optional+Computed: unset in config preserves backend tags so
  out-of-band attachments don't churn plans; explicitly set means Terraform
  owns the field and reverts manual UI edits; `tags = []` clears the entity's
  tags. Backend distinguishes `null`/absent (keep) from `[]` (clear) on the
  wire and the provider honors that.
- **`streamkap_tags` data source.** Lists tags filtered by `filter_name`,
  `filter_type`, or `filter_ids`. Routes through `GET /tags`, falls back to
  `POST /tags/search` automatically when `filter_ids` is large enough that the
  URL would risk a length limit. Use it instead of hardcoding tag IDs.
- **`streamkap_tag.type` plan-time validation.** Element values are now
  validated against the backend `TagTypeEnum` (`environment`, `general`,
  `sources`, `destinations`, `pipelines`, `transforms`, `topics`, `services`,
  `users`, `tenant`). Typos fail at plan, not after a round-trip.
- **Two previously-unwired source connectors** (backend exposed them, the
  provider didn't): `streamkap_source_salesforce_webhook`,
  `streamkap_source_zendesk_webhook`.

### Fixed
- **`streamkap_topic` Read silently no-op'd against the API response.** The
  internal API struct mapped `topic_id` and a top-level `partition_count`, but
  the backend's `GET /topics/{id}` returns `id` and `kafka.partitions.count` —
  so every Read decoded into an empty struct and the resource only looked
  unchanged because prior state survived. `Topic.TopicID` now decodes from
  `id`, and `partition_count` is lifted from `kafka.partitions.count` after
  Unmarshal.
- **`streamkap_tag` adopt-on-exists.** When `CreateTag` returns "A tag with
  identical properties already exists" (e.g. a leaked test fixture or a
  parallel-apply race), the provider now looks the tag up by name via the
  list endpoint and adopts it instead of failing the apply — matching the
  existing behavior for sources, destinations, and transforms.
- **`streamkap_pipeline` — "planned set element does not correlate with any element in actual" on `transforms[].topics`.**
  When a transform attached to a pipeline had been deployed, the backend stamps its
  `topic_ids` with a `transform_<id>_<version>` prefix
  (`python-be-streamkap` `app/utils/api/v2/api_transforms_utils.py:427`). The pipeline
  read path then swapped that internal id into the Terraform state in place of the
  pretty topic name, so the next apply saw the user's pretty name in the plan and
  the full topic_id in state and aborted. State now always carries the pretty
  `topic` field returned by the backend, matching what users write in
  `transforms[*].topics` (e.g. `<schema>.<table>`).
- **`streamkap_transform_*` — `replay_window` left Unknown after Create.**
  `replay_window` is `Optional+Computed` with no default and the backend never
  echoes it back, so on initial Create with no prior state the framework's
  `UseStateForUnknown` plan modifier could not resolve the planned Unknown,
  leaving Terraform to error with "still indicated an unknown value." Create
  now concretizes the field to Null when the user did not set it.

---

## [3.0.0-beta.1] - 2026-03-04 (Pre-release)

> **Beta Release** — This is a pre-release version intended for testing and early feedback.
> It is **not recommended for production use**. If you are currently on v2.x, there is no
> urgency to upgrade. Wait for the stable v3.0.0 release unless you specifically need the
> new resources below and are comfortable with potential breaking changes in future betas.
>
> Install the beta explicitly:
> ```hcl
> version = "3.0.0-beta.1"
> ```

### Added
- **`streamkap_destination_weaviate` resource** - Weaviate vector database destination connector
- **`streamkap_kafka_user` resource** - Kafka user management with ACL-based topic access control
- **`streamkap_client_credential` resource** - API token (client credential) management for machine-to-machine authentication
- **`streamkap_roles` data source** - List available roles for client credential assignment
- **Data source tests** - Acceptance tests for all 6 data sources (tag, topic, topics, topic_metrics, transform, roles)
- **Schema backward compatibility snapshots** - Coverage for all 54 resources (was 16, now complete)
- **AGENTS.md** - AI coding agent guide following emerging standards for AI-assisted development
- **Provider Configuration Validation Tests** - Tests for missing credentials, empty values, environment variable fallbacks
- **Example File Validation Tests** - Tests to verify all example .tf files are valid HCL and have required files
- **429 Rate Limit Handling** - Retry logic now handles HTTP 429 (Too Many Requests) responses

### Changed
- **GNUmakefile** - Enhanced with comprehensive build, test, lint, and development targets including:
  - `make test` - Run unit tests
  - `make test-schema` - Run schema compatibility tests
  - `make test-validators` - Run validator tests
  - `make test-integration` - Run VCR integration tests
  - `make testacc` - Run acceptance tests
  - `make lint` - Run golangci-lint
  - `make cassettes` - Record VCR cassettes
  - `make snapshots` - Update schema snapshots
  - `make sweep` - Clean up orphaned test resources
  - `make validate-examples` - Validate example Terraform files

---

## [2.1.19+] - 2026-02-05 (Development)

### Added
- **Comprehensive Test Suite**
  - Acceptance tests for all 14 new source connectors (AlloyDB, DB2, DocumentDB, Elasticsearch, MariaDB, MongoDB Hosted, Oracle, Oracle AWS, PlanetScale, Redis, S3, Supabase, Vitess, Webhook)
  - Acceptance tests for all 15 new destination connectors (Azure Blob, BigQuery, CockroachDB, DB2, GCS, HTTP Sink, Kafka Direct, Motherduck, MySQL, Oracle, R2, Redis, Redshift, SQL Server, Starburst)
  - Acceptance tests for 3 additional transforms (EnrichAsync, FanOut, Rollup)
  - Migration tests for backward compatibility validation
  - Smoke tests for connectors without live credentials (Oracle, BigQuery, Redshift, Starburst, Motherduck)
  - Negative tests for API error handling (401, 404, 422, 5xx)
  - Schema validation tests for required fields, enums, and sliders
  - State conflict tests for drift detection and external modifications
  - Test sweepers for sources, destinations, transforms, and pipelines
- **Documentation**
  - Comprehensive testing documentation (`docs/TESTING.md`)
  - Code generator documentation (`docs/CODE_GENERATOR.md`)
  - Architecture comparison audit report (`docs/audits/architecture-comparison-report.md`)
  - Entity config schema audit (`docs/audits/entity-config-schema-audit.md`)
  - Backend code reference guide (`docs/audits/backend-code-reference.md`)
- **Examples**
  - Examples for Weaviate destination
  - All 20 sources, 23 destinations, and 6 transforms now have basic.tf and complete.tf examples
- **API Client Enhancements**
  - ListSources, ListDestinations, ListTransforms, ListPipelines methods for sweepers
- **Full Connector Coverage** - All 42 connectors now exposed:
  - **14 new source connectors**: AlloyDB, DB2, DocumentDB, Elasticsearch, MariaDB, MongoDB Hosted, Oracle, Oracle AWS, PlanetScale, Redis, S3, Supabase, Vitess, Webhook
  - **15 new destination connectors**: Azure Blob, BigQuery, CockroachDB, DB2, GCS, HTTP Sink, Kafka Direct, Motherduck, MySQL, Oracle, R2, Redis, Redshift, SQL Server, Starburst
  - Total: 20 sources + 22 destinations
- **AI-Agent Compatibility (Terraform MCP Server)**
  - All resources now have both `Description` (plain text) and `MarkdownDescription` (rich text)
  - Enum fields list all valid values in descriptions
  - Default values are documented in field descriptions
  - Sensitive fields include security notes
- Basic and complete example configurations for all resources
  - `basic.tf` - minimal required configuration
  - `complete.tf` - all available options with comments
- Examples for all 6 transform types (previously missing)
- Example for tag resource (previously missing)
- Resource-level documentation with links to Streamkap docs
- Configurable timeouts for all resources (create, update, delete operations)
- Automatic retry with exponential backoff for transient errors
- Unit tests for helper functions and retry logic

### Changed
- Updated inline GoDoc comments in `internal/resource/connector/base.go`, `cmd/tfgen/parser.go`, `cmd/tfgen/generator.go`
- Added comprehensive architecture documentation for three-layer pattern (Generated Schemas → Thin Wrappers → Shared Base Resource)
- Improved README with accurate connector counts (20 sources, 23 destinations, 8 transforms)
- Regenerated all connector schemas with comprehensive descriptions
- Restructured examples directory with `basic.tf` and `complete.tf` patterns
- Improved error handling with retry for transient failures
- Default timeouts: create/update/delete 20m

### Fixed
- Fixed typo "Tranform" → "Transform" in transform data source
- Fixed incorrect attribute names in destination examples (S3, Iceberg, PostgreSQL)
- Fixed pipeline example output reference bug
- Added missing `sensitive = true` to ClickHouse password variable

### Technical Details
- tfgen code generator now outputs both Description and MarkdownDescription
- Backend investigation confirmed KC timeout is 15s per server
- Conservative retry strategy: 3 attempts, 10s minimum delay
- Retry only on mutating operations (not reads)

## [2.0.0] - 2026-01-09

### Added
- **Transform Resources**: 6 new resource types for data transformation
  - `streamkap_transform_map_filter` - Filter and transform records
  - `streamkap_transform_enrich` - Enrich data with external sources
  - `streamkap_transform_enrich_async` - Async enrichment transforms
  - `streamkap_transform_sql_join` - SQL-based join transforms
  - `streamkap_transform_rollup` - Aggregation/rollup transforms
  - `streamkap_transform_fan_out` - Fan-out transforms for multiple outputs
- **Destination Kafka**: New Kafka destination connector
- **Destination Iceberg**: New Iceberg destination connector
- Comprehensive migration guide (`docs/MIGRATION.md`)
- Architecture documentation (`docs/ARCHITECTURE.md`)
- Developer guide (`docs/DEVELOPMENT.md`)

### Changed
- **Code Generation Architecture**: Provider now uses config-driven code generation
  - BaseConnectorResource pattern for sources and destinations
  - BaseTransformResource pattern for transforms
  - Generated schemas from backend configuration files
- Standardized attribute naming across all connectors

### Deprecated
- PostgreSQL source attributes (will be removed in v3.0):
  - `insert_static_key_field_1` → use `transforms_insert_static_key1_static_field`
  - `insert_static_key_value_1` → use `transforms_insert_static_key1_static_value`
  - `insert_static_value_field_1` → use `transforms_insert_static_value1_static_field`
  - `insert_static_value_1` → use `transforms_insert_static_value1_static_value`
  - `insert_static_key_field_2` → use `transforms_insert_static_key2_static_field`
  - `insert_static_key_value_2` → use `transforms_insert_static_key2_static_value`
  - `insert_static_value_field_2` → use `transforms_insert_static_value2_static_field`
  - `insert_static_value_2` → use `transforms_insert_static_value2_static_value`
  - `predicates_istopictoenrich_pattern` → use `predicates_is_topic_to_enrich_pattern`
- Snowflake destination attributes (will be removed in v3.0):
  - `auto_schema_creation` → use `create_schema_auto`

### Breaking Changes
- **PostgreSQL Source**:
  - `database_port`: Type changed from `Int64` to `String`
  - `signal_data_collection_schema_or_database`: Now required (was optional)
- **Snowflake Destination**:
  - `auto_qa_dedupe_table_mapping`: Type changed from `Map` to `String`

### Removed
- Legacy connector implementation files (replaced by generated code)

## [1.x.x] - Previous Releases

See [GitHub Releases](https://github.com/streamkap-com/terraform-provider-streamkap/releases) for previous versions.
