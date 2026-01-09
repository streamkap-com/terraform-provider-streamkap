# Backend Code Reference Guide

This document provides a comprehensive reference to the Streamkap Python FastAPI backend codebase. Use this when debugging API integration issues, understanding endpoint behavior, or validating Terraform provider implementation.

**Backend Repository**: `/Users/alexandrubodea/Documents/Repositories/python-be-streamkap`
**OpenAPI Specification**: `https://api.streamkap.com/openapi.json`

---

## Table of Contents

1. [Repository Structure](#1-repository-structure)
2. [API Endpoints](#2-api-endpoints)
3. [Entity Models](#3-entity-models)
4. [Plugin Architecture](#4-plugin-architecture)
5. [Dynamic Configuration Resolution](#5-dynamic-configuration-resolution)
6. [CRUD Operations](#6-crud-operations)
7. [Authentication & Multi-Tenancy](#7-authentication--multi-tenancy)
8. [Error Handling](#8-error-handling)
9. [Key Patterns](#9-key-patterns)
10. [Debugging Guide](#10-debugging-guide)

---

## 1. Repository Structure

### 1.1 Top-Level Structure

```
python-be-streamkap/
├── app/
│   ├── api/                  # FastAPI route definitions
│   ├── models/               # Pydantic models
│   │   ├── api/             # Request/Response models
│   │   ├── database/        # Database models
│   │   └── base/            # Base model classes
│   ├── sources/             # Source connector plugins
│   │   └── plugins/         # Per-connector configuration
│   ├── destinations/        # Destination connector plugins
│   │   └── plugins/         # Per-connector configuration
│   ├── transforms/          # Transform plugins
│   │   └── plugins/         # Per-transform configuration
│   ├── services/            # Business logic services
│   ├── repositories/        # Database access layer
│   └── utils/               # Utility functions
└── tests/                   # Test files
```

### 1.2 Key Directories for Terraform Integration

| Directory | Purpose | Terraform Relevance |
|-----------|---------|---------------------|
| `app/api/` | API endpoints | Understand request/response formats |
| `app/models/api/` | Pydantic models | Validate data structures |
| `app/sources/plugins/` | Source configurations | Schema generation |
| `app/destinations/plugins/` | Destination configurations | Schema generation |
| `app/transforms/plugins/` | Transform configurations | Schema generation |
| `app/utils/entity_changes.py` | CRUD logic | Understand create/update flow |

---

## 2. API Endpoints

### 2.1 Sources API

**File**: `app/api/sources_api.py`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/sources` | List sources (paginated) |
| POST | `/sources/search` | Search sources (POST for large filters) |
| GET | `/sources/brief` | List all sources (brief) |
| GET | `/sources/connectors` | List available connector types |
| POST | `/sources` | Create new source |
| PUT | `/sources/{id}` | Update source |
| DELETE | `/sources/{id}` | Delete source |
| POST | `/sources/{id}/actions/{action}` | Execute action (pause/resume/restart) |

**Key Query Parameters**:
- `secret_returned=true` - Include sensitive fields in response
- `unwind_topics=false` - Don't expand topic details

### 2.2 Destinations API

**File**: `app/api/destinations_api.py`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/destinations` | List destinations (paginated) |
| POST | `/destinations/search` | Search destinations (POST) |
| GET | `/destinations/brief` | List all destinations (brief) |
| GET | `/destinations/connectors` | List available connector types |
| POST | `/destinations` | Create new destination |
| PUT | `/destinations/{id}` | Update destination |
| DELETE | `/destinations/{id}` | Delete destination |

### 2.3 Transforms API

**File**: `app/api/transforms_api.py`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/transforms` | List transforms |
| POST | `/transforms` | Create transform |
| PUT | `/transforms/{id}` | Update transform |
| DELETE | `/transforms/{id}` | Delete transform |

### 2.4 Pipelines API

**File**: `app/api/pipelines_api.py`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/pipelines` | List pipelines |
| POST | `/pipelines` | Create pipeline |
| PUT | `/pipelines/{id}` | Update pipeline |
| DELETE | `/pipelines/{id}` | Delete pipeline |

### 2.5 Topics API

**File**: `app/api/topics_api.py`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/topics` | List topics |
| GET | `/topics/{id}` | Get topic details |
| PUT | `/topics/{id}` | Update topic |

### 2.6 Tags API

**File**: `app/api/tags_api.py`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tags` | List tags |
| GET | `/tags?tag_ids={id}` | Get tag by ID |

---

## 3. Entity Models

### 3.1 Source Connector Model

**File**: `app/models/api/sources/common.py`

```python
class SourceConnector(StreamkapBase):
    name: str | None = None
    connector: str | None = None              # Connector code (e.g., "postgresql")
    id: str | None = None                     # MongoDB ObjectId as string
    connector_display_name: str | None = None
    created_timestamp: str | datetime | None = None
    sub_id: str | None = None
    tenant_id: str | None = None
    service_id: str | None = None
    config: Dict[str | int, Union[str, int, bool, None]] | None = None
    topic_ids: List[str | int] | None = None
    topic_map: Dict[str | int, List[str | int]] | None = None
    topics: List[str | int] | None = None
    tasks: List[int] | None = None
    connector_status: str | None = None       # "Active", "Paused", "Broken", etc.
    task_statuses: Dict | None = None
```

### 3.2 Destination Connector Model

**File**: `app/models/api/destinations/common.py`

```python
class DestinationConnector(StreamkapBase):
    name: str | None = None
    connector: str | None = None
    id: str | None = None
    connector_display_name: str | None = None
    created_timestamp: str | datetime | None = None
    sub_id: str | None = None
    tenant_id: str | None = None
    config: Dict[str | int, Union[str, int, bool, None, list, dict]] | None = None
    topic_ids: List[str | int] | None = None
    topic_map: Dict[str | int, List[str | int]] | None = None
    topics: List[str | int] | None = None
    tasks: List[int] | None = None
    connector_status: str | None = None
    task_statuses: Dict | None = None
```

### 3.3 Create/Update Request Models

**File**: `app/models/api/sources/common.py`

```python
class CreateSourceReq(StreamkapBase):
    name: str
    connector: str                            # Required: connector code
    config: Dict[str, Any]                    # Configuration values
    created_from: EntityOriginEnum | None = None  # "terraform", "ui", etc.
```

### 3.4 Entity Origin Enum

**File**: `app/models/api/app_base.py`

```python
class EntityOriginEnum(str, Enum):
    terraform = "terraform"
    ui = "ui"
    api = "api"
    # ... other origins
```

### 3.5 Connector Status Values

```python
CONNECTOR_STATUS = [
    "Active",      # Running normally
    "Paused",      # Manually paused
    "Stopped",     # Manually stopped
    "Broken",      # Error state
    "Starting",    # Starting up
    "Unassigned",  # No Kafka Connect worker assigned
    "Unknown"      # Status unknown
]
```

---

## 4. Plugin Architecture

### 4.1 Plugin Directory Structure

```
app/sources/plugins/postgresql/
├── configuration.latest.json    # Schema definition
├── dynamic_utils.py            # Dynamic value resolution
└── __init__.py
```

### 4.2 Configuration File

**Path**: `app/{entity_type}s/plugins/{connector}/configuration.latest.json`

See [Entity Config Schema Audit](./entity-config-schema-audit.md) for detailed schema documentation.

### 4.3 Dynamic Utils Module

**Path**: `app/{entity_type}s/plugins/{connector}/dynamic_utils.py`

Contains functions referenced by `function_name` in configuration:

```python
# Example from postgresql/dynamic_utils.py

async def get_database_hostname(source):
    """Resolves actual hostname (may be SSH tunnel endpoint)."""
    database_hostname = SqlSrcCfgHelper.get_database_hostname(source)
    # Validation logic...
    return database_hostname

def get_ssh_destination_hostname(source):
    """Returns SSH destination if SSH enabled."""
    return SqlSrcCfgHelper.get_ssh_destination_hostname(source)

def get_database_port(source):
    """Resolves actual port (may be SSH tunnel port)."""
    return SqlSrcCfgHelper.get_database_port(source)
```

### 4.4 Common Dynamic Functions

| Function | Purpose | Used By |
|----------|---------|---------|
| `get_database_hostname` | Resolve hostname (SSH aware) | PostgreSQL, MySQL, etc. |
| `get_database_port` | Resolve port (SSH aware) | PostgreSQL, MySQL, etc. |
| `get_ssh_destination_hostname` | Get SSH target host | All SSH-capable sources |
| `get_ssh_destination_port` | Get SSH target port | All SSH-capable sources |
| `get_table_include_list` | Process table list | All DB sources |
| `get_schema_registry_url` | Get schema registry URL | All connectors |
| `get_connection_url` | Build connection URL | Destinations |
| `get_signal_data_collection_name` | Get signal table | PostgreSQL, MySQL |

---

## 5. Dynamic Configuration Resolution

### 5.1 Resolution Flow

```
User Input → configuration.latest.json → dynamic_utils.py → Final Config
```

1. User provides values for `user_defined: true` fields
2. Backend loads configuration schema
3. For each `type: "dynamic"` field:
   - Calls `function_name` from `dynamic_utils.py`
   - Passes current config values as dependencies
   - Stores computed value

### 5.2 Resolution Code

**File**: `app/utils/fetch_utils.py`

```python
async def resolve_dynamic_config(entity_type, connector, config, entity_id):
    """Resolve all dynamic configuration values."""
    dynamic_utils = get_dynamic_utils(entity_type, connector)

    for field in configuration["config"]:
        if field.get("value", {}).get("type") == "dynamic":
            function_name = field["value"]["function_name"]
            func = getattr(dynamic_utils, function_name)

            # Build context with dependencies
            context = {"entity_id": entity_id, "config": config}

            # Call function (may be async)
            if inspect.iscoroutinefunction(func):
                value = await func(context)
            else:
                value = func(context)

            config[field["name"]] = value
```

### 5.3 Dependency Resolution Order

Fields are resolved based on `dependencies` array:

```json
{
  "name": "connection.url",
  "value": {
    "type": "dynamic",
    "function_name": "get_connection_url",
    "dependencies": [
      "hostname",
      "port",
      "ssl",
      "database"
    ]
  }
}
```

Dependencies must be resolved before the dependent field.

---

## 6. CRUD Operations

### 6.1 Entity Changes Module

**File**: `app/utils/entity_changes.py`

This is the central module for all CRUD operations on sources, destinations, and transforms.

### 6.2 Create Flow

```python
async def upsert_entities(
    input_body,
    entity_type,          # "sources", "destinations", "transforms"
    entity_id,            # None for create, ObjectId for update
    sub_id,
    tenant_id,
    service_id,
    db,
    secret_returned=False,
    user_id=None,
    user_email=None,
):
    # 1. Build entity metadata
    entity_metadata = {
        "sub_id": sub_id,
        "tenant_id": tenant_id,
        "service_id": service_id,
        "name": name,
        "deleted": False,
    }

    # 2. Insert placeholder to get entity_id
    if create_new:
        result = await db_insert_one(entity_type, entity_metadata)
        entity_id = str(result.inserted_id)
        entity_metadata["created_from"] = input_body.get("created_from")

    # 3. Resolve dynamic configuration
    entity_body, connect_body = await get_source_destination_entity_bodies(...)

    # 4. Create/Update Kafka Connect connector
    await upsert_kafka_connector(kafka_connector_name, connect_body, ...)

    # 5. Save to MongoDB
    await db_replace_one(entity_type, {"_id": _id}, entity_body)

    return entity_body
```

### 6.3 `created_from` Handling

**Location**: `app/utils/entity_changes.py:178`

```python
if create_new:
    entity_metadata["created_from"] = input_body.get("created_from")
else:
    entity_metadata["created_from"] = existing_object_mongo.get("created_from")
```

**Terraform Provider Must**:
- Send `created_from: "terraform"` on create
- Preserve existing `created_from` on update (backend handles this)

### 6.4 Delete Flow

```python
async def delete_entities(entity_id, entity_type, tenant_id, service_id):
    # 1. Soft delete in MongoDB
    await db_soft_delete_one(entity_type, {"_id": ObjectId(entity_id)})

    # 2. Delete Kafka Connect connector
    await delete_kafka_connector(kafka_connector_name, ...)

    # 3. Cleanup related resources (topics, SSH tunnels, etc.)
    await cleanup_entity_resources(entity_id, entity_type)
```

---

## 7. Authentication & Multi-Tenancy

### 7.1 Authentication Flow

**File**: `app/utils/api/api_utils.py`

```python
class Authorization:
    def __init__(self, permissions: list[str]):
        self.permissions = permissions

    async def __call__(self, request: Request) -> User:
        # Validate Bearer token
        # Check permissions
        # Return user object
```

### 7.2 Multi-Tenant Context

**File**: `app/utils/api/tenant_infras_utils.py`

```python
def get_tenant_id() -> str:
    """Get current tenant ID from request context."""

def get_service_id() -> str:
    """Get current service ID from request context."""

def get_sub_id() -> str:
    """Get current subscription ID from request context."""
```

### 7.3 Required Headers

| Header | Purpose |
|--------|---------|
| `Authorization` | Bearer token |
| `X-Tenant-Id` | Tenant identifier (usually from token) |

---

## 8. Error Handling

### 8.1 Standard Error Response

```python
class HTTPException:
    status_code: int
    detail: str | dict
```

### 8.2 Common Error Patterns

**Validation Error (400):**
```json
{
  "detail": "Replication Slot Name and Publication Name already exist..."
}
```

**Not Found (404):**
```json
{
  "detail": "Entity not found"
}
```

**Conflict (409):**
```json
{
  "detail": "Entity with this name already exists"
}
```

### 8.3 Error Handling in Terraform Provider

```go
// Check for error detail in response
type APIError struct {
    Detail string `json:"detail"`
}

func (c *Client) handleError(resp *http.Response) error {
    var apiErr APIError
    if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
        return fmt.Errorf("API error: %s", apiErr.Detail)
    }
    return fmt.Errorf("HTTP %d", resp.StatusCode)
}
```

---

## 9. Key Patterns

### 9.1 Config Field Naming

**User-Defined Pattern:**
```
{field}.user.defined → User provides this
{field}             → System computes from user.defined
```

Example:
- `database.hostname.user.defined` - User input
- `database.hostname` - Computed (may be SSH tunnel)

### 9.2 Conditional Fields

Fields can be conditionally required based on other fields:

```json
{
  "name": "ssh.host",
  "conditions": [
    {"operator": "EQ", "config": "ssh.enabled", "value": true}
  ]
}
```

### 9.3 Set-Once Properties

Some fields cannot be changed after creation:

**File**: `app/utils/entity_changes.py:228`

```python
def validate_set_once_properties(
    entity_type,
    connector,
    new_config,
    existing_config,
    create_new,
):
    """Validate that set-once properties aren't modified."""
    # PostgreSQL: slot.name, publication.name
    # MySQL: database.server.id
```

### 9.4 Encrypted Fields

Fields with `encrypt: true` are encrypted in storage:

```python
# app/utils/entity_changes.py
encrypted_fields = [field for field in config if field.get("encrypt")]
for field in encrypted_fields:
    value = entity_body["config"].get(field["name"])
    if value:
        entity_body["config"][field["name"]] = encrypt_value(value)
```

---

## 10. Debugging Guide

### 10.1 API Request Debugging

Check the Terraform provider's debug logs:

```go
tflog.Debug(ctx, fmt.Sprintf(
    "Request details:\n"+
        "\tMethod: %s\n"+
        "\tURL: %s\n"+
        "\tBody: %s",
    req.Method,
    req.URL.String(),
    payload,
))
```

### 10.2 Backend Logs

Backend logs include request/response details:

```python
backend_logger.info("Creating new source")
backend_logger.debug(f"Input body: {input_body}")
backend_logger.error(f"Error: {str(e)}")
```

### 10.3 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "Entity not found" | Wrong ID format | Use string ObjectId |
| "Replication Slot exists" | Duplicate slot name | Use unique slot name |
| "SSH tunnel error" | SSH config invalid | Check SSH host/port/user |
| "401 Unauthorized" | Invalid/expired token | Refresh OAuth token |
| "400 Bad Request" | Missing required field | Check config schema |

### 10.4 Validating Config Against Schema

```python
# Check which fields are required
for field in config:
    if field.get("user_defined") and field.get("required"):
        print(f"Required: {field['name']}")
```

### 10.5 Testing API Locally

```bash
# Get sources
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.streamkap.com/sources?secret_returned=true"

# Create source
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "test", "connector": "postgresql", "config": {...}}' \
  "https://api.streamkap.com/sources?secret_returned=true"
```

---

## Appendix A: File Quick Reference

### API Files

| File | Purpose |
|------|---------|
| `app/api/sources_api.py` | Source endpoints |
| `app/api/destinations_api.py` | Destination endpoints |
| `app/api/transforms_api.py` | Transform endpoints |
| `app/api/pipelines_api.py` | Pipeline endpoints |
| `app/api/topics_api.py` | Topic endpoints |
| `app/api/tags_api.py` | Tag endpoints |

### Model Files

| File | Purpose |
|------|---------|
| `app/models/api/sources/common.py` | Source Pydantic models |
| `app/models/api/destinations/common.py` | Destination Pydantic models |
| `app/models/api/app_base.py` | Base models, EntityOriginEnum |
| `app/models/base/streamkap_base.py` | StreamkapBase class |

### Utility Files

| File | Purpose |
|------|---------|
| `app/utils/entity_changes.py` | CRUD operations |
| `app/utils/fetch_utils.py` | Dynamic config resolution |
| `app/utils/kafka_connect_utils.py` | Kafka Connect API |
| `app/utils/api/tenant_infras_utils.py` | Multi-tenant utilities |

### Plugin Locations

| Type | Path Pattern |
|------|--------------|
| Source config | `app/sources/plugins/{code}/configuration.latest.json` |
| Source dynamic | `app/sources/plugins/{code}/dynamic_utils.py` |
| Destination config | `app/destinations/plugins/{code}/configuration.latest.json` |
| Destination dynamic | `app/destinations/plugins/{code}/dynamic_utils.py` |
| Transform config | `app/transforms/plugins/{code}/configuration.latest.json` |

---

## Appendix B: API Response Examples

### B.1 Get Source Response

```json
{
  "total": 1,
  "page_size": 50,
  "page": 1,
  "result": [
    {
      "id": "64a1b2c3d4e5f6a7b8c9d0e1",
      "name": "my-postgresql-source",
      "connector": "postgresql",
      "connector_display_name": "PostgreSQL",
      "connector_status": "Active",
      "created_timestamp": "2024-01-15T10:30:00Z",
      "config": {
        "database.hostname.user.defined": "db.example.com",
        "database.port.user.defined": "5432",
        "database.user": "streamkap",
        "database.dbname": "mydb",
        "schema.include.list": "public",
        "table.include.list.user.defined": "users,orders",
        "slot.name": "streamkap_slot",
        "publication.name": "streamkap_pub"
      },
      "topic_ids": ["topic_123", "topic_456"],
      "topics": ["source_64a1b2c3.public.users", "source_64a1b2c3.public.orders"],
      "task_statuses": {
        "0": {"state": "RUNNING"}
      }
    }
  ]
}
```

### B.2 Create Source Request

```json
{
  "name": "my-postgresql-source",
  "connector": "postgresql",
  "created_from": "terraform",
  "config": {
    "database.hostname.user.defined": "db.example.com",
    "database.port.user.defined": "5432",
    "database.user": "streamkap",
    "database.password": "secret123",
    "database.dbname": "mydb",
    "schema.include.list": "public",
    "table.include.list.user.defined": "users,orders",
    "slot.name": "streamkap_slot",
    "publication.name": "streamkap_pub",
    "ssh.enabled": false
  }
}
```

---

*Document generated: 2026-01-09*
*Last audit: Terraform Provider Refactor Design v1.0*
