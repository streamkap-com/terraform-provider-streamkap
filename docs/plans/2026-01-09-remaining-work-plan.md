# Remaining Work Plan - Terraform Provider Refactor

**Date**: 2026-01-09
**Status**: Active
**Last Audit**: 2026-01-09 (cross-checked with backend)

## Executive Summary

This plan covers the remaining work after the main refactor (Phases 0-4, 6):
1. ✅ Complete documentation (DONE - needs commit)
2. Implement Transform resources (8 types, 6 active + 2 coming soon)
3. Clean up legacy code (13 files)
4. Additional connectors (14 sources + 15 destinations - FUTURE)

---

## Backend vs Terraform Comparison

### Sources

| Backend (20) | In Terraform | Status |
|--------------|--------------|--------|
| postgresql | ✅ | Migrated |
| mysql | ✅ | Migrated |
| mongodb | ✅ | Migrated |
| dynamodb | ✅ | Migrated |
| sqlserveraws | ✅ (as sqlserver) | Migrated |
| kafkadirect | ✅ | Migrated |
| alloydb | ❌ | Missing |
| db2 | ❌ | Missing |
| documentdb | ❌ | Missing |
| elasticsearch | ❌ | Missing |
| mariadb | ❌ | Missing |
| mongodbhosted | ❌ | Missing |
| oracle | ❌ | Missing |
| oracleaws | ❌ | Missing |
| planetscale | ❌ | Missing |
| redis | ❌ | Missing |
| s3 | ❌ | Missing |
| supabase | ❌ | Missing |
| vitess | ❌ | Missing |
| webhook | ❌ | Missing |

**Implemented: 6/20 (30%)**

### Destinations

| Backend (22) | In Terraform | Status |
|--------------|--------------|--------|
| snowflake | ✅ | Migrated |
| clickhouse | ✅ | Migrated |
| databricks | ✅ | Migrated |
| postgresql | ✅ | Migrated |
| s3 | ✅ | Migrated |
| iceberg | ✅ | Migrated |
| kafka | ✅ | Migrated |
| azblob | ❌ | Missing |
| bigquery | ❌ | Missing |
| cockroachdb | ❌ | Missing |
| db2 | ❌ | Missing |
| gcs | ❌ | Missing |
| httpsink | ❌ | Missing |
| kafkadirect | ❌ | Missing |
| motherduck | ❌ | Missing |
| mysql | ❌ | Missing |
| oracle | ❌ | Missing |
| r2 | ❌ | Missing |
| redis | ❌ | Missing |
| redshift | ❌ | Missing |
| sqlserver | ❌ | Missing |
| starburst | ❌ | Missing |

**Implemented: 7/22 (32%)**

### Transforms

| Backend (8) | In Terraform | Backend Status |
|-------------|--------------|----------------|
| map_filter | ❌ Resource | Active |
| enrich | ❌ Resource | Active |
| enrich_async | ❌ Resource | Active |
| sql_join | ❌ Resource | Active |
| rollup | ❌ Resource | Active |
| fan_out | ❌ Resource | Active |
| toast_handling | ❌ Resource | Coming Soon |
| un_nesting | ❌ Resource | Coming Soon |

**Note:** Transform datasource exists, but NO transform resources.

---

## Current State

### Completed
- ✅ Code generation framework (`cmd/tfgen/`)
- ✅ Generic `BaseConnectorResource` with unified CRUD
- ✅ 6 source connectors migrated
- ✅ 7 destination connectors migrated
- ✅ CI/CD workflows
- ✅ Topics, Pipelines, Tags resources
- ✅ Documentation created (DEVELOPMENT.md, ARCHITECTURE.md, .env.example)

### Outstanding
- ❌ Documentation not committed
- ❌ Transform RESOURCES (currently only datasource exists)
- ❌ Legacy code cleanup (13 files)
- ❌ Additional connectors (29 total missing)

---

## Task 1: Commit Documentation ✅ CREATED

Files created but not committed:
- `docs/DEVELOPMENT.md` - Developer guide
- `docs/ARCHITECTURE.md` - Architecture overview
- `.env.example` - Environment template

**Action:** Review and commit these files.

---

## Task 2: Implement Transform Resources

### 2.1 Transform API Differences (CRITICAL)

Transforms differ significantly from connectors:

| Aspect | Sources/Destinations | Transforms |
|--------|---------------------|------------|
| Type field | `connector` (string) | `transform` (enum) |
| API endpoints | `/sources`, `/destinations` | `/transforms` |
| Delete endpoint | DELETE `/{id}` | DELETE `?id={id}` (query param!) |
| Create payload | `{ connector, name, config }` | `{ transform, config }` |
| Config structure | Flat config map | Dot-notation aliases |
| Implementation | None | Has code/SQL implementation |
| Versioning | No | Yes - history tracking |
| Backend engine | Kafka Connect | Apache Flink |
| Status values | Active, Paused, Stopped, Broken, Starting, Unassigned, Unknown | RUNNING, INITIALIZING, DEPLOYING, CANCELLING, CANCELED, CREATED, RESTARTING, FAILING, FAILED, STOPPED, UNKNOWN |

### 2.2 Transform Request Payload Structure

```go
// CreateTransformRequest for API
type CreateTransformRequest struct {
    Transform   string         `json:"transform"`    // enum: map_filter, enrich, etc.
    Config      map[string]any `json:"config"`       // uses dot-notation keys
    CreatedFrom string         `json:"created_from"` // "terraform"
}

// UpdateTransformRequest for API
type UpdateTransformRequest struct {
    ID          string         `json:"id"`
    Transform   string         `json:"transform"`
    Config      map[string]any `json:"config"`
}

// Transform Response
type Transform struct {
    ID              string         `json:"id"`
    Name            string         `json:"name"`              // from config["transforms.name"]
    TransformType   string         `json:"transform_type"`
    Status          string         `json:"status"`            // computed
    Config          map[string]any `json:"config"`
    Implementation  map[string]any `json:"implementation"`    // type-specific
    Version         int            `json:"version"`
    TopicIDs        []string       `json:"topic_ids"`
    CreatedAt       string         `json:"created_at"`
    UpdatedAt       string         `json:"updated_at"`
}
```

### 2.3 Base Config Fields (all transforms)

```
transforms.name                           # string, required
transforms.language                       # string, optional (PYTHON, JAVASCRIPT, SQL)
transforms.input.topic.pattern            # string, required
transforms.output.topic.pattern           # string, required
transforms.input.serialization.format     # enum, required (AVRO, JSON, etc.)
transforms.output.serialization.format    # enum, required
transforms.input.job.parallelism          # int, default 5
transforms.topic.ttl                      # string, optional
```

### 2.4 API Client Updates

Add to `internal/api/client.go`:
```go
// Transform APIs (extended)
CreateTransform(ctx context.Context, reqPayload CreateTransformRequest) (*Transform, error)
UpdateTransform(ctx context.Context, transformID string, reqPayload UpdateTransformRequest) (*Transform, error)
DeleteTransform(ctx context.Context, transformID string) error
```

Implementation in `internal/api/transform.go`:
- POST `/transforms` for create
- PUT `/transforms` for update (NOT PUT `/transforms/{id}`)
- DELETE `/transforms?id={id}` for delete (query param, NOT path param)

### 2.5 Transform Types (8)

| Transform | Config Path | Active |
|-----------|-------------|--------|
| map_filter | `app/transforms/plugins/map_filter/configuration.latest.json` | ✅ |
| enrich | `app/transforms/plugins/enrich/configuration.latest.json` | ✅ |
| enrich_async | `app/transforms/plugins/enrich_async/configuration.latest.json` | ✅ |
| sql_join | `app/transforms/plugins/sql_join/configuration.latest.json` | ✅ |
| rollup | `app/transforms/plugins/rollup/configuration.latest.json` | ✅ |
| fan_out | `app/transforms/plugins/fan_out/configuration.latest.json` | ✅ |
| toast_handling | `app/transforms/plugins/toast_handling/configuration.latest.json` | ⏳ Coming Soon |
| un_nesting | `app/transforms/plugins/un_nesting/configuration.latest.json` | ⏳ Coming Soon |

### 2.6 Generator Updates

Add support for `--type transform` in `cmd/tfgen/generator.go`:
- Different package output (`internal/generated/transform_*.go`)
- Different model struct (no `connector` field, has `transform_type`)
- Different field mapping format (dot-notation aliases)

### 2.7 BaseTransformResource

Create `internal/resource/transform/base.go`:
- Similar to BaseConnectorResource but for transforms
- Different API calls (CreateTransform, UpdateTransform, DeleteTransform)
- Handle implementation field (type-specific)
- Handle version field (computed)

---

## Task 3: Code Cleanup

### 3.1 Legacy Files to Remove (13 total)

**Sources (6 files):**
- `internal/resource/source/postgresql.go`
- `internal/resource/source/mongodb.go`
- `internal/resource/source/mysql.go`
- `internal/resource/source/dynamodb.go`
- `internal/resource/source/sqlserver.go`
- `internal/resource/source/kafkadirect.go`

**Destinations (7 files):**
- `internal/resource/destination/snowflake.go`
- `internal/resource/destination/clickhouse.go`
- `internal/resource/destination/databricks.go`
- `internal/resource/destination/postgresql.go`
- `internal/resource/destination/s3.go`
- `internal/resource/destination/iceberg.go`
- `internal/resource/destination/kafka.go`

### 3.2 Verification Steps

```bash
# Before deletion - verify build
go build ./...

# Delete legacy files
rm internal/resource/source/{postgresql,mongodb,mysql,dynamodb,sqlserver,kafkadirect}.go
rm internal/resource/destination/{snowflake,clickhouse,databricks,postgresql,s3,iceberg,kafka}.go

# After deletion - verify build and tests
go build ./...
go test -v -short ./...
```

---

## Task 4: Additional Connectors (FUTURE - Optional)

### Missing Sources (14)
alloydb, db2, documentdb, elasticsearch, mariadb, mongodbhosted,
oracle, oracleaws, planetscale, redis, s3, supabase, vitess, webhook

### Missing Destinations (15)
azblob, bigquery, cockroachdb, db2, gcs, httpsink, kafkadirect,
motherduck, mysql, oracle, r2, redis, redshift, sqlserver, starburst

These can be added incrementally using the generator. Each requires:
1. Generate schema from backend `configuration.latest.json`
2. Create config wrapper implementing `ConnectorConfig`
3. Register in `provider.go`
4. Add acceptance test
5. Add example

---

## Implementation Order

### Phase A - Commit Documentation (5 min)
- [x] Create DEVELOPMENT.md
- [x] Create ARCHITECTURE.md
- [x] Create .env.example
- [ ] Review and commit

### Phase B - Transform API Client (30 min)
- [ ] Update Transform struct with full fields
- [ ] Add CreateTransformRequest, UpdateTransformRequest types
- [ ] Implement CreateTransform (POST /transforms)
- [ ] Implement UpdateTransform (PUT /transforms - note: no ID in path)
- [ ] Implement DeleteTransform (DELETE /transforms?id={id})
- [ ] Add to StreamkapAPI interface

### Phase C - Generator Transform Support (30 min)
- [ ] Add `--type transform` support
- [ ] Generate transform schemas from backend configs
- [ ] Output to `internal/generated/transform_*.go`

### Phase D - Transform Resources (1 hour)
- [ ] Create `internal/resource/transform/base.go` (BaseTransformResource)
- [ ] Create config wrappers for 6 active transforms (defer toast_handling, un_nesting)
- [ ] Register in provider.go

### Phase E - Cleanup (15 min)
- [ ] Remove 13 legacy files
- [ ] Verify build
- [ ] Run tests
- [ ] Commit

---

## Success Criteria

- [ ] Documentation committed
- [ ] 6 active Transform resources implemented and registered
- [ ] Legacy code (13 files) removed
- [ ] All unit tests pass
- [ ] Code compiles cleanly
- [ ] At least one acceptance test passes

---

## Files to Create/Modify

### Create
- `internal/generated/transform_*.go` (6 files for active transforms)
- `internal/resource/transform/base.go`
- `internal/resource/transform/*_generated.go` (6 files)

### Modify
- `internal/api/client.go` - Add transform CRUD interface methods
- `internal/api/transform.go` - Implement CRUD, update structs
- `internal/provider/provider.go` - Register transform resources
- `cmd/tfgen/generator.go` - Support transform type

### Delete
- 13 legacy connector files

### Commit
- `docs/DEVELOPMENT.md`
- `docs/ARCHITECTURE.md`
- `.env.example`
- `docs/plans/2026-01-09-remaining-work-plan.md`
