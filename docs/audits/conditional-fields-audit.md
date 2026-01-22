# Conditional Fields Audit Report

*Generated: 2026-01-22 09:12:29*

This report identifies fields in backend configuration that have conditional requirements
(i.e., they are conditionally required based on the value of other fields).

**Fields with conditions should be marked as `Optional` in Terraform**, not `Required`,
because the Terraform provider cannot evaluate backend conditional logic at plan time.

---

## Executive Summary

### Quick Stats

| Metric | Count |
|--------|-------|
| Total connectors with conditional fields | 32 |
| Total conditional fields in backend | 159 |
| Connectors with Terraform/Backend mismatches | 9 |
| **Fields needing change in Terraform** | **13** |

### Action Required

The following Terraform schema files have fields marked as `Required: true` that should be
`Optional: true` because they have conditional requirements in the backend:

- `internal/generated/source_alloydb.go`
- `internal/generated/source_elasticsearch.go`
- `internal/generated/source_mariadb.go`
- `internal/generated/source_mysql.go`
- `internal/generated/source_oracle.go`
- `internal/generated/source_oracleaws.go`
- `internal/generated/source_postgresql.go`
- `internal/generated/source_redis.go`
- `internal/generated/source_supabase.go`

---

## Terraform/Backend Mismatches (Action Required)

These fields have `conditions` in the backend config but are currently marked as
`Required: true` in Terraform. **They must be changed to `Optional: true`.**

### Source: alloydb

*File: `internal/generated/source_alloydb.go`*

#### `signal_data_collection_schema_or_database`

- **Backend name**: `signal.data.collection.schema.or.database`
- **Display name**: Signal Table Schema
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "snapshot.read.only.user.defined",
    "value": "No"
  }
]
```

*Condition: `snapshot.read.only.user.defined` = `No`*

---

#### `heartbeat_data_collection_schema_or_database`

- **Backend name**: `heartbeat.data.collection.schema.or.database`
- **Display name**: Heartbeat Table Database
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "heartbeat.enabled",
    "value": true
  },
  {
    "operator": "EQ",
    "config": "snapshot.read.only.user.defined",
    "value": "No"
  }
]
```

*Condition: `heartbeat.enabled` = `True` AND `snapshot.read.only.user.defined` = `No`*

---

### Source: elasticsearch

*File: `internal/generated/source_elasticsearch.go`*

#### `http_auth_user`

- **Backend name**: `http.auth.user`
- **Display name**: Username
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "http.auth.user.defined",
    "value": "Basic"
  }
]
```

*Condition: `http.auth.user.defined` = `Basic`*

---

#### `http_auth_password`

- **Backend name**: `http.auth.password`
- **Display name**: Password
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "http.auth.user.defined",
    "value": "Basic"
  }
]
```

*Condition: `http.auth.user.defined` = `Basic`*

---

### Source: mariadb

*File: `internal/generated/source_mariadb.go`*

#### `heartbeat_data_collection_schema_or_database`

- **Backend name**: `heartbeat.data.collection.schema.or.database`
- **Display name**: Heartbeat Table Database
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "heartbeat.enabled",
    "value": true
  },
  {
    "operator": "EQ",
    "config": "snapshot.gtid",
    "value": "No"
  }
]
```

*Condition: `heartbeat.enabled` = `True` AND `snapshot.gtid` = `No`*

---

### Source: mysql

*File: `internal/generated/source_mysql.go`*

#### `heartbeat_data_collection_schema_or_database`

- **Backend name**: `heartbeat.data.collection.schema.or.database`
- **Display name**: Heartbeat Table Database
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "heartbeat.enabled",
    "value": true
  },
  {
    "operator": "EQ",
    "config": "snapshot.gtid",
    "value": "No"
  }
]
```

*Condition: `heartbeat.enabled` = `True` AND `snapshot.gtid` = `No`*

---

### Source: oracle

*File: `internal/generated/source_oracle.go`*

#### `heartbeat_data_collection_schema_or_database`

- **Backend name**: `heartbeat.data.collection.schema.or.database`
- **Display name**: Heartbeat Table Database
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "heartbeat.enabled",
    "value": true
  }
]
```

*Condition: `heartbeat.enabled` = `True`*

---

### Source: oracleaws

*File: `internal/generated/source_oracleaws.go`*

#### `heartbeat_data_collection_schema_or_database`

- **Backend name**: `heartbeat.data.collection.schema.or.database`
- **Display name**: Heartbeat Table Database
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "heartbeat.enabled",
    "value": true
  }
]
```

*Condition: `heartbeat.enabled` = `True`*

---

### Source: postgresql

*File: `internal/generated/source_postgresql.go`*

#### `signal_data_collection_schema_or_database`

- **Backend name**: `signal.data.collection.schema.or.database`
- **Display name**: Signal Table Schema
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "snapshot.read.only.user.defined",
    "value": "No"
  }
]
```

*Condition: `snapshot.read.only.user.defined` = `No`*

---

#### `heartbeat_data_collection_schema_or_database`

- **Backend name**: `heartbeat.data.collection.schema.or.database`
- **Display name**: Heartbeat Table Database
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "heartbeat.enabled",
    "value": true
  },
  {
    "operator": "EQ",
    "config": "snapshot.read.only.user.defined",
    "value": "No"
  }
]
```

*Condition: `heartbeat.enabled` = `True` AND `snapshot.read.only.user.defined` = `No`*

---

### Source: redis

*File: `internal/generated/source_redis.go`*

#### `redis_stream_name`

- **Backend name**: `redis.stream.name`
- **Display name**: Stream Name
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "connector.class.type",
    "value": "Stream"
  }
]
```

*Condition: `connector.class.type` = `Stream`*

---

### Source: supabase

*File: `internal/generated/source_supabase.go`*

#### `signal_data_collection_schema_or_database`

- **Backend name**: `signal.data.collection.schema.or.database`
- **Display name**: Signal Table Schema
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "snapshot.read.only.user.defined",
    "value": "No"
  }
]
```

*Condition: `snapshot.read.only.user.defined` = `No`*

---

#### `heartbeat_data_collection_schema_or_database`

- **Backend name**: `heartbeat.data.collection.schema.or.database`
- **Display name**: Heartbeat Table Database
- **Current Terraform**: `Required: true`
- **Should be**: `Optional: true, Computed: true`
- **Backend required (when condition met)**: `True`
- **Default value**: `None`

**Conditions** (field is only required when):

```json
[
  {
    "operator": "EQ",
    "config": "heartbeat.enabled",
    "value": true
  },
  {
    "operator": "EQ",
    "config": "snapshot.read.only.user.defined",
    "value": "No"
  }
]
```

*Condition: `heartbeat.enabled` = `True` AND `snapshot.read.only.user.defined` = `No`*

---

## All Backend Conditional Fields

Complete list of all fields with conditions from backend configs, organized by connector type.

### Sources

#### alloydb

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `signal_data_collection_schema_or_database` | `signal.data.collection.schema.or.database` | **Yes** | `-` | snapshot.read.only.user.defined=No |
| `column_include_list_user_defined` | `column.include.list.user.defined` | No | `-` | column.include.list.toggled=True |
| `column_exclude_list_user_defined` | `column.exclude.list.user.defined` | No | `-` | column.include.list.toggled=False |
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `-` | heartbeat.enabled=True, snapshot.read.only.user.defined=No |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### db2

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### documentdb

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### elasticsearch

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `http_auth_user` | `http.auth.user` | **Yes** | `-` | http.auth.user.defined=Basic |
| `http_auth_password` | `http.auth.password` | **Yes** | `-` | http.auth.user.defined=Basic |

#### kafkadirect

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `schemas_enable` | `schemas.enable` | **Yes** | `-` | format=json |

#### mariadb

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `signal_data_collection_schema_or_database` | `signal.data.collection.schema.or.database` | No | `-` | snapshot.gtid=No |
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `-` | heartbeat.enabled=True, snapshot.gtid=No |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### mongodb

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### mongodbhosted

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### mysql

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `signal_data_collection_schema_or_database` | `signal.data.collection.schema.or.database` | No | `-` | snapshot.gtid=No |
| `column_include_list_user_defined` | `column.include.list.user.defined` | No | `-` | column.include.list.toggled=True |
| `column_exclude_list_user_defined` | `column.exclude.list.user.defined` | No | `-` | column.include.list.toggled=False |
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `-` | heartbeat.enabled=True, snapshot.gtid=No |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### oracle

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `-` | heartbeat.enabled=True |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### oracleaws

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `-` | heartbeat.enabled=True |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### planetscale

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### postgresql

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `signal_data_collection_schema_or_database` | `signal.data.collection.schema.or.database` | **Yes** | `-` | snapshot.read.only.user.defined=No |
| `column_include_list_user_defined` | `column.include.list.user.defined` | No | `-` | column.include.list.toggled=True |
| `column_exclude_list_user_defined` | `column.exclude.list.user.defined` | No | `-` | column.include.list.toggled=False |
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `-` | heartbeat.enabled=True, snapshot.read.only.user.defined=No |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### redis

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `redis_stream_name` | `redis.stream.name` | **Yes** | `-` | connector.class.type=Stream |
| `redis_stream_offset_user_defined` | `redis.stream.offset.user.defined` | No | `Latest` | connector.class.type=Stream |
| `redis_stream_delivery_user_defined` | `redis.stream.delivery.user.defined` | No | `At Least Once` | connector.class.type=Stream |
| `redis_stream_block_seconds_user_defined` | `redis.stream.block.seconds.user.defined` | No | `1` | connector.class.type=Stream |
| `redis_stream_consumer_group` | `redis.stream.consumer.group` | No | `kafka-consumer-group` | connector.class.type=Stream |
| `redis_stream_consumer_name_user_defined` | `redis.stream.consumer.name.user.defined` | No | `consumer` | connector.class.type=Stream |
| `redis_keys_pattern_user_defined` | `redis.keys.pattern.user.defined` | **Yes** | `*` | connector.class.type=Keys |
| `redis_keys_timeout_seconds_user_defined` | `redis.keys.timeout.seconds.user.defined` | No | `300` | connector.class.type=Keys |
| `mode` | `mode` | No | `LIVE` | connector.class.type=Keys |
| `topic_use_stream_name` | `topic.use.stream.name` | No | `-` | connector.class.type=Stream |
| `topic_user_defined` | `topic.user.defined` | **Yes** | `-` | None=None |
| `tasks_max` | `tasks.max` | **Yes** | `1` | connector.class.type=Stream |

#### sqlserveraws

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `streamkap` | heartbeat.enabled=True |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### supabase

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `signal_data_collection_schema_or_database` | `signal.data.collection.schema.or.database` | **Yes** | `-` | snapshot.read.only.user.defined=No |
| `column_include_list_user_defined` | `column.include.list.user.defined` | No | `-` | column.include.list.toggled=True |
| `column_exclude_list_user_defined` | `column.exclude.list.user.defined` | No | `-` | column.include.list.toggled=False |
| `heartbeat_data_collection_schema_or_database` | `heartbeat.data.collection.schema.or.database` | **Yes** | `-` | heartbeat.enabled=True, snapshot.read.only.user.defined=No |
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### vitess

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

### Destinations

#### azblob

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `format_csv_write_headers` | `format.csv.write.headers` | No | `-` | format.user.defined=csv |
| `compression` | `compression` | No | `-` | format.user.defined=['json', 'avro', 'parquet', 'binary', 'csv'] |

#### clickhouse

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `hard_delete` | `hard.delete` | No | `True` | ingestion.mode=upsert |

#### cockroachdb

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `primary_key_mode_user_defined` | `primary.key.mode.user.defined` | No | `record_key` | delete.enabled=False |
| `primary_key_fields` | `primary.key.fields` | No | `-` | primary.key.mode.user.defined=['record_key', 'record_value'] |
| `transforms_changeTopicName_match_regex_user_defined` | `transforms.changeTopicName.match.regex.user.defined` | No | `-` | topic2table.map.user.defined=False |
| `transforms_changeTopicName_mapping` | `transforms.changeTopicName.mapping` | No | `-` | topic2table.map.user.defined=True |

#### databricks

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `transforms_changeTopicName_match_regex_user_defined` | `transforms.changeTopicName.match.regex.user.defined` | No | `-` | topic2table.map.user.defined=False |
| `transforms_changeTopicName_mapping` | `transforms.changeTopicName.mapping` | No | `-` | topic2table.map.user.defined=True |

#### db2

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `primary_key_mode_user_defined` | `primary.key.mode.user.defined` | No | `record_key` | delete.enabled=False |
| `primary_key_fields` | `primary.key.fields` | No | `-` | primary.key.mode.user.defined=['record_key', 'record_value'] |

#### httpsink

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `http_headers_authorization` | `http.headers.authorization` | No | `-` | http.authorization.type=static |
| `oauth2_access_token_url` | `oauth2.access.token.url` | No | `-` | http.authorization.type=oauth2 |
| `oauth2_client_id` | `oauth2.client.id` | No | `-` | http.authorization.type=oauth2 |
| `oauth2_client_secret` | `oauth2.client.secret` | No | `-` | http.authorization.type=oauth2 |
| `oauth2_scope` | `oauth2.scope` | No | `-` | http.authorization.type=oauth2 |
| `batch_max_size` | `batch.max.size` | No | `500` | batching.enabled=True |
| `batch_buffering_enabled` | `batch.buffering.enabled` | No | `-` | batching.enabled=True |
| `batch_max_time_ms` | `batch.max.time.ms` | No | `10000` | batching.enabled=True, batch.buffering.enabled=True |
| `batch_prefix` | `batch.prefix` | No | `[` | batching.enabled=True |
| `batch_suffix` | `batch.suffix` | No | `]` | batching.enabled=True |
| `batch_separator` | `batch.separator` | No | `,` | batching.enabled=True |
| `errors_tolerance_user_defined` | `errors.tolerance.user.defined` | No | `none` | batching.enabled=False |

#### iceberg

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `iceberg_catalog_name` | `iceberg.catalog.name` | No | `-` | iceberg.catalog.type=['rest', 'hive'] |
| `iceberg_catalog_client_assume-role_arn` | `iceberg.catalog.client.assume-role.arn` | No | `-` | iceberg.catalog.type=['glue'] |
| `iceberg_catalog_uri` | `iceberg.catalog.uri` | No | `-` | iceberg.catalog.type=['rest', 'hive'] |
| `iceberg_catalog_s3_access-key-id` | `iceberg.catalog.s3.access-key-id` | No | `-` | iceberg.catalog.type=['rest', 'hive'] |
| `iceberg_catalog_s3_secret-access-key` | `iceberg.catalog.s3.secret-access-key` | No | `-` | iceberg.catalog.type=['rest', 'hive'] |
| `iceberg_tables_default-id-columns` | `iceberg.tables.default-id-columns` | No | `-` | insert.mode.user.defined=upsert |

#### kafka

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `json_schema_enable` | `json.schema.enable` | No | `-` | destination.format=json |
| `schema_registry_url_user_defined` | `schema.registry.url.user.defined` | No | `-` | destination.format=avro |

#### motherduck

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `transforms_changeTopicName_match_regex_user_defined` | `transforms.changeTopicName.match.regex.user.defined` | No | `-` | topic2table.map.user.defined=False |
| `transforms_changeTopicName_mapping` | `transforms.changeTopicName.mapping` | No | `-` | topic2table.map.user.defined=True |

#### mysql

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `primary_key_mode_user_defined` | `primary.key.mode.user.defined` | No | `record_key` | delete.enabled=False |
| `primary_key_fields` | `primary.key.fields` | No | `-` | primary.key.mode.user.defined=['record_key', 'record_value'] |
| `transforms_changeTopicName_match_regex_user_defined` | `transforms.changeTopicName.match.regex.user.defined` | No | `-` | topic2table.map.user.defined=False |
| `transforms_changeTopicName_mapping` | `transforms.changeTopicName.mapping` | No | `-` | topic2table.map.user.defined=True |

#### oracle

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `primary_key_mode_user_defined` | `primary.key.mode.user.defined` | No | `record_key` | delete.enabled=False |
| `primary_key_fields` | `primary.key.fields` | No | `-` | primary.key.mode.user.defined=['record_key', 'record_value'] |
| `transforms_changeTopicName_match_regex_user_defined` | `transforms.changeTopicName.match.regex.user.defined` | No | `-` | topic2table.map.user.defined=False |
| `transforms_changeTopicName_mapping` | `transforms.changeTopicName.mapping` | No | `-` | topic2table.map.user.defined=True |

#### postgresql

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `ssh_host` | `ssh.host` | No | `-` | ssh.enabled=True |
| `ssh_port` | `ssh.port` | No | `22` | ssh.enabled=True |
| `ssh_user` | `ssh.user` | No | `streamkap` | ssh.enabled=True |
| `primary_key_mode_user_defined` | `primary.key.mode.user.defined` | No | `record_key` | delete.enabled=False |
| `primary_key_fields` | `primary.key.fields` | No | `-` | primary.key.mode.user.defined=['record_key', 'record_value'] |
| `transforms_changeTopicName_match_regex_user_defined` | `transforms.changeTopicName.match.regex.user.defined` | No | `-` | topic2table.map.user.defined=False |
| `transforms_changeTopicName_mapping` | `transforms.changeTopicName.mapping` | No | `-` | topic2table.map.user.defined=True |
| `ssh_public_key_user_displayed` | `ssh.public.key.user.displayed` | No | `<SSH.PUBLIC.KEY>` | ssh.enabled=True |

#### snowflake

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `snowflake_private_key_passphrase` | `snowflake.private.key.passphrase` | No | `-` | snowflake.private.key.passphrase.secured=True |
| `hard_delete` | `hard.delete` | No | `True` | ingestion.mode=upsert |
| `schema_evolution` | `schema.evolution` | No | `basic` | ingestion.mode=upsert |
| `use_hybrid_tables` | `use.hybrid.tables` | No | `-` | ingestion.mode=upsert |
| `apply_dynamic_table_script` | `apply.dynamic.table.script` | No | `-` | ingestion.mode=append |
| `create_sql_execute` | `create.sql.execute` | No | `CREATE OR REPLACE DYNAMIC TABL` | ingestion.mode=append, apply.dynamic.table.script=True |
| `sql_table_name` | `sql.table.name` | No | `{{table}}_DT` | ingestion.mode=append, apply.dynamic.table.script=True |
| `create_sql_data` | `create.sql.data` | No | `-` | ingestion.mode=append, apply.dynamic.table.script=True |
| `auto_qa_dedupe_table_mapping` | `auto.qa.dedupe.table.mapping` | No | `-` | ingestion.mode=append, apply.dynamic.table.script=True |

#### sqlserver

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `primary_key_mode_user_defined` | `primary.key.mode.user.defined` | No | `record_key` | delete.enabled=False |
| `primary_key_fields` | `primary.key.fields` | No | `-` | primary.key.mode.user.defined=['record_key', 'record_value'] |
| `transforms_changeTopicName_match_regex_user_defined` | `transforms.changeTopicName.match.regex.user.defined` | No | `-` | topic2table.map.user.defined=False |
| `transforms_changeTopicName_mapping` | `transforms.changeTopicName.mapping` | No | `-` | topic2table.map.user.defined=True |

#### weaviate

| Terraform Field | Backend Field | Required | Default | Conditions |
|-----------------|---------------|----------|---------|------------|
| `weaviate_api_key` | `weaviate.api.key` | No | `-` | weaviate.auth.scheme=API_KEY |
| `weaviate_oidc_client_secret` | `weaviate.oidc.client.secret` | No | `-` | weaviate.auth.scheme=OIDC_CLIENT_CREDENTIALS |
| `weaviate_oidc_scopes` | `weaviate.oidc.scopes` | No | `openid` | weaviate.auth.scheme=OIDC_CLIENT_CREDENTIALS |
| `document_id_field_name` | `document.id.field.name` | No | `id` | document.id.strategy.user.defined=Field ID |
| `vector_field_name` | `vector.field.name` | No | `vector` | vector.strategy.user.defined=Field Vector |
