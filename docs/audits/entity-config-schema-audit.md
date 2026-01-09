# Entity Configuration JSON Schema Audit

This document provides a comprehensive audit of the `configuration.latest.json` schema structure used by Streamkap backend for sources, destinations, and transforms. This is the authoritative reference for the code generator when creating typed Terraform resource schemas.

**Backend Repository**: `/Users/alexandrubodea/Documents/Repositories/python-be-streamkap`

---

## Table of Contents

1. [Schema Overview](#1-schema-overview)
2. [Top-Level Schema Structure](#2-top-level-schema-structure)
3. [Config Entry Schema](#3-config-entry-schema)
4. [Value Object Schema](#4-value-object-schema)
5. [Control Types Reference](#5-control-types-reference)
6. [Dynamic Value Resolution](#6-dynamic-value-resolution)
7. [Conditional Visibility](#7-conditional-visibility)
8. [Entity Type Differences](#8-entity-type-differences)
9. [Sources Reference](#9-sources-reference)
10. [Destinations Reference](#10-destinations-reference)
11. [Transforms Reference](#11-transforms-reference)
12. [Terraform Mapping Rules](#12-terraform-mapping-rules)

---

## 1. Schema Overview

Each connector/transform plugin has a `configuration.latest.json` file that defines:
- Display metadata (name, description)
- Configuration fields with validation rules
- Dynamic value resolution functions
- Conditional field visibility

**Location Pattern**:
- Sources: `app/sources/plugins/{connector_code}/configuration.latest.json`
- Destinations: `app/destinations/plugins/{connector_code}/configuration.latest.json`
- Transforms: `app/transforms/plugins/{transform_type}/configuration.latest.json`

---

## 2. Top-Level Schema Structure

### 2.1 Sources/Destinations (Connectors)

```json
{
  "display_name": "PostgreSQL",           // Human-readable name
  "schema_levels": ["schema", "table"],   // Optional: schema hierarchy levels
  "debezium_connector_name": "postgresql", // Optional: Debezium connector identifier
  "serialisation": "io.confluent...",     // Optional: Kafka serialization class
  "metrics": [...],                       // Optional: metrics definitions (sources only)
  "config": [...]                         // Array of config entry objects
}
```

### 2.2 Transforms

```json
{
  "display_name": "Transform/Filter Records",
  "description": "Apply custom JavaScript or Python logic...",
  "config": [...]                         // Array of config entry objects
}
```

**Key Differences**:
- Sources have: `schema_levels`, `debezium_connector_name`, `serialisation`, `metrics`
- Destinations have: `metrics` (usually empty)
- Transforms have: `description` (no `schema_levels` or `metrics`)

---

## 3. Config Entry Schema

Each entry in the `config` array follows this structure:

```json
{
  "name": "database.hostname.user.defined",  // Internal config key
  "description": "PostgreSQL Hostname...",   // Optional: help text
  "user_defined": true,                      // CRITICAL: true = Terraform field
  "required": true,                          // Optional: field requirement
  "order_of_display": 1,                     // Optional: UI ordering
  "display_name": "Hostname",                // Optional: UI label
  "value": {...},                            // Value definition object
  "tab": "auth",                             // Optional: UI tab grouping
  "kafka_config": true,                      // Optional: passed to Kafka Connect
  "encrypt": true,                           // Optional: field is sensitive
  "display_advanced": true,                  // Optional: advanced settings
  "conditions": [...],                       // Optional: conditional visibility
  "schema_level": "table",                   // Optional: schema selector type
  "schema_name_format": "<schema>.<table>",  // Optional: format pattern
  "ssh_update_determinant": true,            // Optional: triggers SSH update
  "show_last": 4,                            // Optional: mask all but last N chars
  "not_clonable": true,                      // Optional: exclude from cloning
  "global": true,                            // Optional: global setting flag
  "set_once": true                           // Optional: field can only be set on creation
}
```

### 3.1 Critical Field: `user_defined`

**This is the most important field for Terraform schema generation:**

| Value | Meaning | Terraform Action |
|-------|---------|------------------|
| `true` | User provides this value | Generate as schema attribute |
| `false` | System computes this value | Skip or mark as computed |

**Only fields with `user_defined: true` should become Terraform schema attributes.**

### 3.2 Field Sensitivity: `encrypt`

When `encrypt: true`, the field should be marked as `Sensitive: true` in Terraform schema.

Common encrypted fields:
- `database.password`
- `aws.secret.key`
- `snowflake.private.key`
- `databricks.token`
- `ssh.private.key`

### 3.3 Kafka Config Flag: `kafka_config`

| Value | Meaning |
|-------|---------|
| `true` | Passed to Kafka Connect configuration |
| `false` | Used only in backend processing |
| `{dynamic}` | Dynamically determines inclusion |

### 3.4 Set Once Fields: `set_once`

When `set_once: true`, the field can only be set during resource creation and cannot be changed afterward.

**Terraform Implication**: Use `RequiresReplace()` plan modifier to force resource recreation when this field changes.

Currently used by:
- `ingestion.mode` in Snowflake destination

---

## 4. Value Object Schema

The `value` object defines the field's type, validation, and defaults:

```json
{
  "control": "string",           // UI control type (see Section 5)
  "type": "raw",                 // Value type: "raw" or "dynamic"
  "default": "5432",             // Default value
  "raw_value": "...",            // Static value (when type=raw, user_defined=false)
  "raw_values": ["opt1", "opt2"], // Options for select controls
  "function_name": "get_...",    // Dynamic resolution function
  "dependencies": ["field1"],    // Dependencies for dynamic resolution
  "max": 100,                    // Slider: maximum value
  "min": 1,                      // Slider: minimum value
  "step": 1,                     // Slider: step increment
  "rows": 15,                    // Textarea: display rows
  "placeholder": "eg. value",    // Input placeholder text
  "readonly": true,              // Read-only field
  "multiline": true,             // Textarea: allow multiline input
  "early_resolved": "fn_name",   // Early resolution function
  "validation": {...}            // Custom validation rules
}
```

### 4.1 Static vs Dynamic Values

**Static (`type: "raw"`):**
```json
{
  "type": "raw",
  "raw_value": "com.streamkap.sink.clickhouse.ClickhouseSinkConnector"
}
```

**Dynamic (`type: "dynamic"`):**
```json
{
  "type": "dynamic",
  "function_name": "get_database_hostname",
  "dependencies": ["database.hostname.user.defined", "ssh.enabled", "ssh.host"]
}
```

**User-Defined with Default:**
```json
{
  "control": "string",
  "default": "5432"
}
```

---

## 5. Control Types Reference

### 5.1 Control Type to Terraform Type Mapping

| Control | Terraform Type | Notes |
|---------|---------------|-------|
| `string` | `types.String` | Basic string input |
| `password` | `types.String` + Sensitive | Masked string input |
| `textarea` | `types.String` | Multi-line string |
| `number` | `types.Int64` or `types.String` | Numeric input (often stored as string) |
| `boolean` | `types.Bool` | True/false toggle |
| `toggle` | `types.Bool` | On/off switch |
| `one-select` | `types.String` | Single selection dropdown |
| `multi-select` | `types.List[types.String]` | Multiple selection (array of strings) |
| `slider` | `types.Int64` | Numeric slider with min/max/step |
| `json` | `types.String` | JSON content (e.g., credentials file) |
| `datetime` | `types.String` | ISO datetime string picker |

**Frontend-Supported but Not Currently Used in Configs:**
| Control | Terraform Type | Notes |
|---------|---------------|-------|
| `code-editor` | `types.String` | Code editing with syntax highlighting |
| `table-multi-select` | `types.List[types.String]` | Table with multiple selection |
| `table-one-select` | `types.String` | Table with single selection |
| `child-form` | Nested Object | Sub-form for complex structures |
| `autocomplete` | `types.String` | Autocomplete text field |

### 5.2 Control Type Examples

**String Control:**
```json
{
  "name": "database.hostname.user.defined",
  "user_defined": true,
  "value": {
    "control": "string"
  }
}
```

**Password Control:**
```json
{
  "name": "database.password",
  "user_defined": true,
  "value": {
    "control": "password"
  },
  "encrypt": true
}
```

**One-Select Control:**
```json
{
  "name": "ingestion.mode",
  "user_defined": true,
  "value": {
    "control": "one-select",
    "raw_values": ["upsert", "append"],
    "default": "upsert"
  }
}
```

**Slider Control:**
```json
{
  "name": "tasks.max",
  "user_defined": true,
  "value": {
    "control": "slider",
    "max": 10,
    "min": 1,
    "default": 5,
    "step": 1
  }
}
```

**Boolean Control:**
```json
{
  "name": "ssl",
  "user_defined": true,
  "value": {
    "control": "boolean",
    "default": true
  }
}
```

**Toggle Control:**
```json
{
  "name": "ssh.enabled",
  "user_defined": true,
  "value": {
    "control": "toggle",
    "type": "raw",
    "default": false
  }
}
```

**Textarea Control:**
```json
{
  "name": "column.include.list.user.defined",
  "user_defined": true,
  "value": {
    "control": "textarea"
  }
}
```

---

## 6. Dynamic Value Resolution

### 6.1 Dynamic Functions

Dynamic values are computed by Python functions in `dynamic_utils.py`:

```json
{
  "name": "database.hostname",
  "user_defined": false,
  "value": {
    "type": "dynamic",
    "function_name": "get_database_hostname",
    "dependencies": [
      "database.hostname.user.defined",
      "ssh.enabled",
      "ssh.host"
    ]
  }
}
```

**Resolution Location**: `app/{entity_type}s/plugins/{connector}/dynamic_utils.py`

### 6.2 Common Dynamic Patterns

**SSH Hostname Resolution:**
- User provides: `database.hostname.user.defined`
- System computes: `database.hostname` (may be SSH tunnel endpoint)

**Connection URL Assembly:**
- User provides: `hostname`, `port`, `database`, `ssl`
- System computes: `connection.url`

**Schema Registry URLs:**
- Function: `get_schema_registry_url`
- Depends on: `allow.customer.kafka.connect.server`

### 6.3 Terraform Implications

**For code generation:**
- Fields with `user_defined: false` and `type: dynamic` should NOT be Terraform attributes
- The `*.user.defined` suffix pattern indicates the user-facing field
- The computed field without suffix is the resolved value

---

## 7. Conditional Visibility

### 7.1 Conditions Array

Fields can be conditionally visible based on other field values:

```json
{
  "name": "ssh.host",
  "user_defined": true,
  "value": {
    "control": "string"
  },
  "conditions": [
    {
      "operator": "EQ",
      "config": "ssh.enabled",
      "value": true
    }
  ]
}
```

### 7.2 Condition Operators

| Operator | Meaning |
|----------|---------|
| `EQ` | Equal to value |
| `NE` | Not equal to value |
| `IN` | Value in list |

### 7.3 Multiple Conditions

Multiple conditions are ANDed together:

```json
{
  "conditions": [
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
}
```

### 7.4 Terraform Implications

Conditional fields require validators that check:
- Field is required when condition is met
- Field is optional/ignored when condition is not met

Use `schema.ConflictsWith`, `RequiredWith`, or custom validators.

---

## 8. Entity Type Differences

### 8.1 Sources

**Unique Characteristics:**
- Have `schema_levels` for schema/table selection
- Have `debezium_connector_name` for Debezium integration
- Have `serialisation` for Kafka message format
- Have detailed `metrics` array for monitoring
- Common fields: hostname, port, user, password, SSL, SSH tunnel

**Common SSH Fields (most sources):**
- `ssh.enabled` (toggle)
- `ssh.host` (conditional)
- `ssh.port` (conditional, default: 22)
- `ssh.user` (conditional, default: streamkap)
- `ssh.public.key.user.displayed` (readonly)

### 8.2 Destinations

**Unique Characteristics:**
- Usually no `schema_levels`
- Empty `metrics` array (metrics computed differently)
- Common fields: hostname/url, credentials, ingestion mode, schema evolution

**Common Fields:**
- `ingestion.mode`: upsert/append
- `schema.evolution`: basic/none
- `tasks.max`: parallelism slider
- `hard.delete`: delete mode toggle

### 8.3 Transforms

**Unique Characteristics:**
- Have `description` at top level
- No `schema_levels`, `metrics`, or Debezium fields
- Simpler config structure
- Common fields: name, language, input/output patterns

**Common Fields:**
- `transforms.name`: transform name
- `transforms.language`: JavaScript/Python/SQL
- `transforms.input.topic.pattern`: regex for input topics
- `transforms.output.topic.pattern`: output topic pattern

---

## 9. Sources Reference

### 9.1 Available Source Connectors (20 Total)

| Connector Code | Display Name | Config Path |
|---------------|--------------|-------------|
| `alloydb` | AlloyDB | `app/sources/plugins/alloydb/configuration.latest.json` |
| `db2` | DB2 | `app/sources/plugins/db2/configuration.latest.json` |
| `documentdb` | DocumentDB | `app/sources/plugins/documentdb/configuration.latest.json` |
| `dynamodb` | DynamoDB | `app/sources/plugins/dynamodb/configuration.latest.json` |
| `elasticsearch` | Elasticsearch | `app/sources/plugins/elasticsearch/configuration.latest.json` |
| `kafkadirect` | Kafka Direct | `app/sources/plugins/kafkadirect/configuration.latest.json` |
| `mariadb` | MariaDB | `app/sources/plugins/mariadb/configuration.latest.json` |
| `mongodb` | MongoDB | `app/sources/plugins/mongodb/configuration.latest.json` |
| `mongodbhosted` | MongoDB Hosted | `app/sources/plugins/mongodbhosted/configuration.latest.json` |
| `mysql` | MySQL | `app/sources/plugins/mysql/configuration.latest.json` |
| `oracle` | Oracle | `app/sources/plugins/oracle/configuration.latest.json` |
| `oracleaws` | Oracle AWS | `app/sources/plugins/oracleaws/configuration.latest.json` |
| `planetscale` | PlanetScale | `app/sources/plugins/planetscale/configuration.latest.json` |
| `postgresql` | PostgreSQL | `app/sources/plugins/postgresql/configuration.latest.json` |
| `redis` | Redis | `app/sources/plugins/redis/configuration.latest.json` |
| `s3` | S3 | `app/sources/plugins/s3/configuration.latest.json` |
| `sqlserveraws` | SQL Server (AWS) | `app/sources/plugins/sqlserveraws/configuration.latest.json` |
| `supabase` | Supabase | `app/sources/plugins/supabase/configuration.latest.json` |
| `vitess` | Vitess | `app/sources/plugins/vitess/configuration.latest.json` |
| `webhook` | Webhook | `app/sources/plugins/webhook/configuration.latest.json` |

### 9.2 PostgreSQL Source Schema (Reference Example)

**User-Defined Fields (Terraform Attributes):**

| Field Name | Control | Required | Sensitive | Default | Tab |
|-----------|---------|----------|-----------|---------|-----|
| `database.hostname.user.defined` | string | yes | no | - | auth |
| `database.port.user.defined` | string | yes | no | 5432 | auth |
| `database.user` | string | yes | no | - | auth |
| `database.password` | password | yes | yes | - | auth |
| `database.dbname` | string | yes | no | - | auth |
| `schema.include.list` | string | yes | no | - | schema |
| `table.include.list.user.defined` | string | yes | no | - | schema |
| `slot.name` | string | yes | no | - | settings |
| `publication.name` | string | yes | no | - | settings |
| `snapshot.read.only.user.defined` | one-select | yes | no | No | settings |
| `signal.data.collection.schema.or.database` | string | conditional | no | - | settings |
| `ssh.enabled` | toggle | no | no | false | settings |
| `ssh.host` | string | conditional | no | - | settings |
| `ssh.port` | string | conditional | no | 22 | settings |
| `ssh.user` | string | conditional | no | streamkap | settings |
| `binary.handling.mode` | one-select | no | no | bytes | settings |
| `decimal.handling.mode` | one-select | no | no | double | settings |
| `tasks.max` | slider | yes | no | 1 | settings |

---

## 10. Destinations Reference

### 10.1 Available Destination Connectors (23 Total)

| Connector Code | Display Name | Config Path |
|---------------|--------------|-------------|
| `azblob` | Azure Blob | `app/destinations/plugins/azblob/configuration.latest.json` |
| `bigquery` | BigQuery | `app/destinations/plugins/bigquery/configuration.latest.json` |
| `clickhouse` | ClickHouse | `app/destinations/plugins/clickhouse/configuration.latest.json` |
| `cockroachdb` | CockroachDB | `app/destinations/plugins/cockroachdb/configuration.latest.json` |
| `databricks` | Databricks | `app/destinations/plugins/databricks/configuration.latest.json` |
| `db2` | DB2 | `app/destinations/plugins/db2/configuration.latest.json` |
| `gcs` | Google Cloud Storage | `app/destinations/plugins/gcs/configuration.latest.json` |
| `httpsink` | HTTP Sink | `app/destinations/plugins/httpsink/configuration.latest.json` |
| `iceberg` | Iceberg | `app/destinations/plugins/iceberg/configuration.latest.json` |
| `kafka` | Kafka | `app/destinations/plugins/kafka/configuration.latest.json` |
| `kafkadirect` | Kafka Direct | `app/destinations/plugins/kafkadirect/configuration.latest.json` |
| `motherduck` | MotherDuck | `app/destinations/plugins/motherduck/configuration.latest.json` |
| `mysql` | MySQL | `app/destinations/plugins/mysql/configuration.latest.json` |
| `oracle` | Oracle | `app/destinations/plugins/oracle/configuration.latest.json` |
| `postgresql` | PostgreSQL | `app/destinations/plugins/postgresql/configuration.latest.json` |
| `r2` | Cloudflare R2 | `app/destinations/plugins/r2/configuration.latest.json` |
| `redis` | Redis | `app/destinations/plugins/redis/configuration.latest.json` |
| `redshift` | Redshift | `app/destinations/plugins/redshift/configuration.latest.json` |
| `rockset` | Rockset | `app/destinations/plugins/rockset/configuration.latest.json` |
| `s3` | S3 | `app/destinations/plugins/s3/configuration.latest.json` |
| `snowflake` | Snowflake | `app/destinations/plugins/snowflake/configuration.latest.json` |
| `sqlserver` | SQL Server | `app/destinations/plugins/sqlserver/configuration.latest.json` |
| `starburst` | Starburst | `app/destinations/plugins/starburst/configuration.latest.json` |

### 10.2 Snowflake Destination Schema (Reference Example)

**User-Defined Fields (Terraform Attributes):**

| Field Name | Control | Required | Sensitive | Default | Tab |
|-----------|---------|----------|-----------|---------|-----|
| `snowflake.url.name` | string | yes | no | - | auth |
| `snowflake.user.name` | string | yes | no | - | auth |
| `snowflake.private.key` | password | yes | yes | - | auth |
| `snowflake.private.key.passphrase` | password | no | yes | - | auth |
| `snowflake.database.name` | string | yes | no | - | auth |
| `snowflake.schema.name` | string | yes | no | streamkap | auth |
| `snowflake.role.name` | string | no | no | - | auth |
| `ingestion.mode` | one-select | no | no | upsert | settings |
| `hard.delete` | boolean | conditional | no | true | settings |
| `schema.evolution` | one-select | no | no | basic | smt |
| `tasks.max` | slider | yes | no | 5 | settings |

### 10.3 ClickHouse Destination Schema (Reference Example)

**User-Defined Fields (Terraform Attributes):**

| Field Name | Control | Required | Sensitive | Default | Tab |
|-----------|---------|----------|-----------|---------|-----|
| `hostname` | string | yes | no | - | auth |
| `port` | string | yes | no | 8443 | auth |
| `connection.username` | string | yes | no | - | auth |
| `connection.password` | password | yes | yes | - | auth |
| `database` | string | no | no | - | auth |
| `ssl` | boolean | no | no | true | settings |
| `ingestion.mode` | one-select | no | no | upsert | settings |
| `hard.delete` | boolean | conditional | no | true | settings |
| `schema.evolution` | one-select | no | no | basic | smt |
| `tasks.max` | slider | yes | no | 5 | settings |
| `topics.config.map` | textarea | no | no | - | settings |

---

## 11. Transforms Reference

### 11.1 Available Transform Types

| Transform Type | Display Name | Config Path |
|---------------|--------------|-------------|
| `map_filter` | Transform/Filter Records | `app/transforms/plugins/map_filter/configuration.latest.json` |
| `rollup` | Rollup | `app/transforms/plugins/rollup/configuration.latest.json` |
| `enrich` | Enrich | `app/transforms/plugins/enrich/configuration.latest.json` |
| `enrich_async` | Async Enrich | `app/transforms/plugins/enrich_async/configuration.latest.json` |
| `fan_out` | Fan Out | `app/transforms/plugins/fan_out/configuration.latest.json` |
| `sql_join` | SQL Join | `app/transforms/plugins/sql_join/configuration.latest.json` |
| `toast_handling` | TOAST Handling | `app/transforms/plugins/toast_handling/configuration.latest.json` |
| `un_nesting` | Un-Nesting | `app/transforms/plugins/un_nesting/configuration.latest.json` |

### 11.2 Map Filter Transform Schema (Reference Example)

**User-Defined Fields (Terraform Attributes):**

| Field Name | Control | Required | Default | Tab |
|-----------|---------|----------|---------|-----|
| `transforms.name` | string | yes | - | settings |
| `transforms.language` | one-select | yes | JavaScript | settings |
| `transforms.input.topic.pattern` | textarea | yes | - | settings |
| `transforms.output.topic.pattern` | textarea | yes | - | settings |

### 11.3 Rollup Transform Schema (Reference Example)

**User-Defined Fields (Terraform Attributes):**

| Field Name | Control | Required | Default | Tab |
|-----------|---------|----------|---------|-----|
| `transforms.name` | string | yes | - | settings |
| `transforms.language` | one-select | yes | SQL | settings |
| `transforms.input.topic.pattern` | textarea | yes | - | settings |
| `transforms.output.topic.pattern` | textarea | yes | - | settings |
| `transforms.input.serialization.format` | one-select | yes | Any | settings |
| `transforms.output.serialization.format` | one-select | yes | Any | settings |

---

## 12. Terraform Mapping Rules

### 12.1 Field Selection Rules

**Include in Terraform Schema:**
1. `user_defined: true`
2. NOT a computed internal field

**Exclude from Terraform Schema:**
1. `user_defined: false`
2. `type: "dynamic"` (computed by backend)
3. `kafka_config: true` only fields (internal)

### 12.2 Type Mapping Rules

```go
// Control type to Terraform type mapping
func controlToTerraformType(control string, encrypt bool) attr.Type {
    switch control {
    case "string", "password", "textarea":
        return types.StringType
    case "number":
        return types.Int64Type  // or StringType if stored as string
    case "boolean", "toggle":
        return types.BoolType
    case "one-select":
        return types.StringType
    case "slider":
        return types.Int64Type
    default:
        return types.StringType
    }
}
```

### 12.3 Validator Generation Rules

**Required Fields:**
```go
schema.StringAttribute{
    Required: true,  // when config.required == true
}
```

**Optional with Default:**
```go
schema.StringAttribute{
    Optional: true,
    Computed: true,
    Default:  stringdefault.StaticString("5432"),
}
```

**One-Select Validation:**
```go
schema.StringAttribute{
    Validators: []validator.String{
        stringvalidator.OneOf("upsert", "append"),
    },
}
```

**Slider Validation:**
```go
schema.Int64Attribute{
    Validators: []validator.Int64{
        int64validator.Between(1, 10),
    },
}
```

**Conditional Required:**
```go
schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        // Required when ssh.enabled == true
        stringvalidator.RequiredWhen(
            path.MatchRelative().AtParent().AtName("ssh_enabled"),
            types.BoolValue(true),
        ),
    },
}
```

### 12.4 Sensitive Field Rules

Mark as sensitive when:
- `encrypt: true`
- `control: "password"`

```go
schema.StringAttribute{
    Sensitive: true,
}
```

### 12.5 Computed Field Rules

Fields computed by backend (read-only):
- `connector_status`
- `task_statuses`
- `topics`
- `topic_ids`

```go
schema.StringAttribute{
    Computed: true,
}
```

---

## Appendix A: Full Config Entry JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "description": "Internal config key name"
    },
    "description": {
      "type": "string",
      "description": "Help text for the field"
    },
    "user_defined": {
      "type": "boolean",
      "description": "true = user provides value, false = system computes"
    },
    "required": {
      "type": "boolean",
      "description": "Whether field is required"
    },
    "order_of_display": {
      "type": "integer",
      "description": "UI ordering"
    },
    "display_name": {
      "type": "string",
      "description": "UI label"
    },
    "value": {
      "type": "object",
      "properties": {
        "control": {
          "type": "string",
          "enum": ["string", "password", "textarea", "number", "boolean", "toggle", "one-select", "slider"]
        },
        "type": {
          "type": "string",
          "enum": ["raw", "dynamic"]
        },
        "default": {},
        "raw_value": {},
        "raw_values": {
          "type": "array"
        },
        "function_name": {
          "type": "string"
        },
        "dependencies": {
          "type": "array",
          "items": {"type": "string"}
        },
        "max": {"type": "number"},
        "min": {"type": "number"},
        "step": {"type": "number"}
      }
    },
    "tab": {
      "type": "string",
      "enum": ["auth", "schema", "settings", "smt"]
    },
    "kafka_config": {
      "oneOf": [
        {"type": "boolean"},
        {"type": "object"}
      ]
    },
    "encrypt": {
      "type": "boolean"
    },
    "display_advanced": {
      "type": "boolean"
    },
    "conditions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "operator": {"type": "string", "enum": ["EQ", "NE", "IN"]},
          "config": {"type": "string"},
          "value": {}
        }
      }
    }
  },
  "required": ["name", "user_defined", "value"]
}
```

---

## Appendix B: Naming Conventions

### B.1 User-Defined Suffix Pattern

Many fields follow the pattern where the user-facing field has a `.user.defined` suffix:

| User Field | Computed Field |
|-----------|---------------|
| `database.hostname.user.defined` | `database.hostname` |
| `database.port.user.defined` | `database.port` |
| `connection.url.user.defined` | `connection.url` |
| `table.include.list.user.defined` | `table.include.list` |
| `databricks.catalog.user.defined` | - (used directly) |

### B.2 Terraform Attribute Naming

Transform backend field names to Terraform-friendly names:
- Replace `.` with `_`
- Remove `.user.defined` suffix
- Use snake_case

Examples:
- `database.hostname.user.defined` → `database_hostname`
- `snowflake.url.name` → `snowflake_url_name`
- `tasks.max` → `tasks_max`

---

*Document generated: 2026-01-09*
*Last audit: Terraform Provider Refactor Design v1.0*
