## 0.1.0 (Unreleased)

FEATURES:

* **PostgreSQL source**: New optional `heartbeat_use_logical_message` (bool,
  default `false`). When `heartbeat_enabled = true` and this is set, the
  connector runs `SELECT pg_logical_emit_message(true, ...)` on each beat to
  advance the replication slot — works on PG14+ primaries with a SELECT-only
  role and is compatible with read-only mode. No `streamkap_heartbeat` table
  or write grant required on the source. Resolves ENG-2398.
