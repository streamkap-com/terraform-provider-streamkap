## 0.1.0 (Unreleased)

FEATURES:

* **PostgreSQL/MySQL sources**: Document Kafka-only heartbeat mode. Setting
  `heartbeat_enabled = true` while leaving `heartbeat_data_collection_schema_or_database`
  unset/null now keeps the connector polling on low-traffic sources without
  requiring a `streamkap_heartbeat` table or write grant in the source DB.
  No schema change — the field has always been optional in the provider; this
  just lights up an existing path that the backend used to reject.
