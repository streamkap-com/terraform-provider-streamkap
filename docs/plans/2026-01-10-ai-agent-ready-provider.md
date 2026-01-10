# AI-Agent Ready Provider Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the Terraform provider AI-agent ready by adding comprehensive descriptions to all schemas so AI assistants (Claude, Cursor, Copilot) generate correct configurations via the Terraform MCP Server.

**Architecture:** Enhance the `tfgen` code generator to output both `Description` and `MarkdownDescription` fields with rich content (valid values, defaults, security notes). Update hand-coded resources (pipeline, tag, topic) and data sources (tag, transform) manually. Improve examples with basic.tf/complete.tf structure.

**Tech Stack:** Go, Terraform Plugin Framework, tfgen code generator

**Key insight from Terraform MCP Server:** The server "exposes the exact schema for every provider resource and data source: argument types, required/optional fields, nested block shapes, computed attributes, deprecated fields" directly from the Terraform Registry. Both `Description` (plain text) and `MarkdownDescription` (rich text) should be present.

---

## Task 1: Add MarkdownDescription Field to FieldData Struct

**Files:**
- Modify: `cmd/tfgen/generator.go` (FieldData struct around line 66)

**Context:** Before updating the template, we need the FieldData struct to have a separate MarkdownDescription field. Currently it only has Description.

**Step 1: Update the FieldData struct**

Find the `FieldData` struct and add `MarkdownDescription` after `Description`:

```go
// FieldData holds data for a single field in the model and schema.
type FieldData struct {
	// Model fields
	GoFieldName string // e.g., "DatabaseHostname"
	GoType      string // e.g., "types.String"
	TfsdkTag    string // e.g., "database_hostname"

	// Schema fields
	TfAttrName          string // e.g., "database_hostname"
	SchemaAttrType      string // e.g., "schema.StringAttribute"
	Required            bool
	Optional            bool
	Computed            bool
	Sensitive           bool
	Description         string // Plain text description for CLI
	MarkdownDescription string // Rich markdown description for docs/AI
	HasDefault          bool
	DefaultFunc         string // e.g., "stringdefault.StaticString(\"5432\")"
	HasValidators       bool
	Validators          string // e.g., "stringvalidator.OneOf(\"Yes\", \"No\")"
	NeedsPlanMod        bool   // needs UseStateForUnknown plan modifier
	RequiresReplace     bool   // needs RequiresReplace plan modifier (set_once)
	IsListType          bool   // needs ElementType for ListAttribute

	// Field mapping
	APIFieldName string // e.g., "database.hostname.user.defined"
}
```

**Step 2: Update entryToFieldData to initialize MarkdownDescription**

Find the `entryToFieldData` function and after setting `field.Description`, add:

```go
// Initialize MarkdownDescription from Description
field.MarkdownDescription = field.Description
```

**Step 3: Run tests**

Run: `go test ./cmd/tfgen/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/tfgen/generator.go
git commit -m "feat(tfgen): add MarkdownDescription field to FieldData struct

- Separate Description (plain text) and MarkdownDescription (rich markdown)
- Prepare for enhanced description generation"
```

---

## Task 2: Update tfgen Template to Output Both Description Fields

**Files:**
- Modify: `cmd/tfgen/generator.go` (schemaTemplate around line 408)

**Context:** The template currently only outputs `MarkdownDescription`. We need both.

**Step 1: Update the template for attribute descriptions**

Find this section in the template:
```go
{{- if .Description }}
				MarkdownDescription: {{ printf "%q" .Description }},
{{- end }}
```

Replace with:
```go
{{- if .Description }}
				Description:         {{ printf "%q" .Description }},
				MarkdownDescription: {{ printf "%q" .MarkdownDescription }},
{{- end }}
```

**Step 2: Update the template for Schema-level description**

Find this line:
```go
		MarkdownDescription: "{{ .DisplayName }} {{ .EntityType }} connector",
```

Replace with:
```go
		Description:         "Manages a {{ .DisplayName }} {{ .EntityType }} connector.",
		MarkdownDescription: "Manages a **{{ .DisplayName }} {{ .EntityType }} connector**.\n\n" +
			"This resource creates and manages a {{ .DisplayName }} {{ .EntityType }} for Streamkap data pipelines.\n\n" +
			"[Documentation](https://docs.streamkap.com/{{ .EntityType }}s/{{ .ConnectorCode }})",
```

**Step 3: Run tests**

Run: `go test ./cmd/tfgen/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/tfgen/generator.go
git commit -m "feat(tfgen): output both Description and MarkdownDescription in template

- Template now outputs both fields for CLI and docs/AI tools
- Resource-level descriptions enhanced with links and formatting
- Follows Terraform best practices for provider documentation"
```

---

## Task 3: Enhance Descriptions with Valid Values for Enum Fields

**Files:**
- Modify: `cmd/tfgen/generator.go` (entryToFieldData function)

**Context:** When a field has a OneOf validator (enum), the valid values should be listed in both descriptions.

**Step 1: Update the one-select handling section**

Find the section that handles validators for one-select fields:
```go
// Handle validators for one-select fields
if entry.Value.Control == "one-select" && len(entry.GetRawValues()) > 0 {
	field.HasValidators = true
	field.Validators = g.oneOfValidator(entry)
}
```

Replace with:
```go
// Handle validators for one-select fields
if entry.Value.Control == "one-select" && len(entry.GetRawValues()) > 0 {
	field.HasValidators = true
	field.Validators = g.oneOfValidator(entry)

	// Enhance descriptions with valid values
	values := entry.GetRawValues()
	valuesStr := strings.Join(values, ", ")
	field.Description = field.Description + " Valid values: " + valuesStr + "."

	// Build markdown list for MarkdownDescription
	var mdValues []string
	for _, v := range values {
		mdValues = append(mdValues, fmt.Sprintf("`%s`", v))
	}
	field.MarkdownDescription = field.MarkdownDescription + " Valid values: " + strings.Join(mdValues, ", ") + "."
}
```

**Step 2: Run tests**

Run: `go test ./cmd/tfgen/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/tfgen/generator.go
git commit -m "feat(tfgen): include valid values in descriptions for enum fields

- Plain text: appends 'Valid values: x, y, z.'
- Markdown: appends 'Valid values: \`x\`, \`y\`, \`z\`.'
- AI agents can now see allowed values in descriptions"
```

---

## Task 4: Enhance Descriptions with Default Values

**Files:**
- Modify: `cmd/tfgen/generator.go` (entryToFieldData function)

**Context:** When a field has a default value, it should be mentioned in the description.

**Step 1: Update the defaults handling section**

Find the section that handles defaults and sets HasDefault. After `field.NeedsPlanMod = false`, add:

```go
// Capture default value and enhance descriptions
switch entry.TerraformType() {
case TerraformTypeString:
	defaultVal := entry.GetDefaultString()
	field.Description = field.Description + fmt.Sprintf(" Defaults to %q.", defaultVal)
	field.MarkdownDescription = field.MarkdownDescription + fmt.Sprintf(" Defaults to `%s`.", defaultVal)
case TerraformTypeInt64:
	defaultVal := entry.GetDefaultInt64()
	field.Description = field.Description + fmt.Sprintf(" Defaults to %d.", defaultVal)
	field.MarkdownDescription = field.MarkdownDescription + fmt.Sprintf(" Defaults to `%d`.", defaultVal)
case TerraformTypeBool:
	defaultVal := entry.GetDefaultBool()
	field.Description = field.Description + fmt.Sprintf(" Defaults to %t.", defaultVal)
	field.MarkdownDescription = field.MarkdownDescription + fmt.Sprintf(" Defaults to `%t`.", defaultVal)
}
```

**Step 2: Run tests**

Run: `go test ./cmd/tfgen/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/tfgen/generator.go
git commit -m "feat(tfgen): include default values in descriptions

- Appends 'Defaults to X.' for fields with default values
- AI agents can now see default behavior in descriptions"
```

---

## Task 5: Add Security Notes for Sensitive Fields

**Files:**
- Modify: `cmd/tfgen/generator.go` (entryToFieldData function)

**Context:** Sensitive fields should have a security note in their description.

**Step 1: Add security notes at the end of entryToFieldData**

Before the `return field` statement, add:

```go
// Add security note for sensitive fields
if field.Sensitive {
	field.Description = field.Description + " This value is sensitive and will not appear in logs or CLI output."
	field.MarkdownDescription = field.MarkdownDescription + "\n\n**Security:** This value is marked sensitive and will not appear in CLI output or logs."
}
```

**Step 2: Run tests**

Run: `go test ./cmd/tfgen/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/tfgen/generator.go
git commit -m "feat(tfgen): add security notes for sensitive fields

- Appends security warning to sensitive field descriptions
- AI agents will inform users about sensitive data handling"
```

---

## Task 6: Regenerate All Schemas with Enhanced Descriptions

**Files:**
- Regenerate: `internal/generated/*.go`

**Context:** After updating tfgen, regenerate all connector schemas to apply the new description format.

**Step 1: Run the code generator**

Run: `go run ./cmd/tfgen/...`
Expected: All files in `internal/generated/` updated

**Step 2: Verify the generated code compiles**

Run: `go build ./...`
Expected: BUILD SUCCESS

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: PASS

**Step 4: Inspect a generated file**

Open `internal/generated/source_postgresql.go` and verify:
- Schema has both `Description` and `MarkdownDescription`
- Fields with validators have valid values listed
- Fields with defaults have defaults mentioned
- Sensitive fields have security notes

**Step 5: Commit**

```bash
git add internal/generated/
git commit -m "feat(generated): regenerate schemas with enhanced descriptions

- All resources now have both Description and MarkdownDescription
- Enum fields list valid values in descriptions
- Default values documented in descriptions
- Sensitive fields have security notes"
```

---

## Task 7: Update Pipeline Resource with Enhanced Descriptions

**Files:**
- Modify: `internal/resource/pipeline/pipeline.go`

**Context:** Pipeline is a hand-coded resource, not generated. Update it with comprehensive descriptions.

**Step 1: Update the Schema function**

Replace the schema definition with enhanced descriptions. The schema should include:

```go
resp.Schema = schema.Schema{
	Description: "Manages a Streamkap pipeline that connects sources to destinations with optional transforms.",
	MarkdownDescription: "Manages a **Streamkap pipeline** that connects sources to destinations with optional transforms.\n\n" +
		"A pipeline is the core orchestration resource that defines how data flows from a source " +
		"connector through optional transforms to a destination connector.\n\n" +
		"[Documentation](https://docs.streamkap.com/pipelines)",
	// ... all attributes with both Description and MarkdownDescription
}
```

Ensure every attribute has both Description and MarkdownDescription with:
- Clear explanations
- Default values mentioned where applicable
- Cross-references to other resources (e.g., "Reference an existing transform resource using `streamkap_transform_*.id`")

**Step 2: Verify compilation**

Run: `go build ./...`
Expected: BUILD SUCCESS

**Step 3: Run tests**

Run: `go test ./internal/resource/pipeline/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/resource/pipeline/pipeline.go
git commit -m "feat(pipeline): enhance schema descriptions for AI readability

- Add both Description and MarkdownDescription to all attributes
- Include default values in descriptions
- Add resource-level documentation link"
```

---

## Task 8: Update Tag Resource with Enhanced Descriptions

**Files:**
- Modify: `internal/resource/tag/tag.go`

**Context:** Tag is a hand-coded resource. Update it with comprehensive descriptions.

**Step 1: Update the Schema function**

```go
resp.Schema = schema.Schema{
	Description: "Manages a Streamkap tag for organizing and categorizing resources.",
	MarkdownDescription: "Manages a **Streamkap tag** for organizing and categorizing resources.\n\n" +
		"Tags can be applied to sources, destinations, and pipelines to help organize " +
		"and filter resources in the Streamkap UI.\n\n" +
		"[Documentation](https://docs.streamkap.com/tags)",
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:         "Unique identifier for the tag. Generated by Streamkap on creation.",
			MarkdownDescription: "Unique identifier for the tag. Generated by Streamkap on creation.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Description:         "Display name of the tag. This is shown in the Streamkap UI.",
			MarkdownDescription: "Display name of the tag. This is shown in the Streamkap UI.",
			Required:            true,
		},
		"description": schema.StringAttribute{
			Description:         "Optional description of the tag's purpose.",
			MarkdownDescription: "Optional description of the tag's purpose.",
			Optional:            true,
		},
		"type": schema.ListAttribute{
			Description:         "List of entity types this tag can be applied to. Valid values: sources, destinations, pipelines.",
			MarkdownDescription: "List of entity types this tag can be applied to. Valid values: `sources`, `destinations`, `pipelines`.",
			Required:            true,
			ElementType:         types.StringType,
		},
		"system": schema.BoolAttribute{
			Description:         "Whether this is a system-managed tag. System tags cannot be modified. Read-only.",
			MarkdownDescription: "Whether this is a system-managed tag. System tags cannot be modified. **Read-only**.",
			Computed:            true,
		},
		"custom": schema.BoolAttribute{
			Description:         "Whether this is a custom (user-created) tag. Read-only.",
			MarkdownDescription: "Whether this is a custom (user-created) tag. **Read-only**.",
			Computed:            true,
		},
	},
}
```

**Step 2: Verify compilation and run tests**

Run: `go build ./... && go test ./internal/resource/tag/... -v`
Expected: BUILD SUCCESS, PASS

**Step 3: Commit**

```bash
git add internal/resource/tag/tag.go
git commit -m "feat(tag): enhance schema descriptions for AI readability

- Add both Description and MarkdownDescription to all attributes
- Include valid values for type field
- Add resource-level documentation link"
```

---

## Task 9: Update Topic Resource with Enhanced Descriptions

**Files:**
- Modify: `internal/resource/topic/topic.go`

**Context:** Topic is a hand-coded resource with minimal descriptions ("Topic ID", "Partition Count"). Update with comprehensive descriptions.

**Step 1: Update the Schema function**

```go
resp.Schema = schema.Schema{
	Description: "Manages a Streamkap Kafka topic's partition count.",
	MarkdownDescription: "Manages a **Streamkap Kafka topic's partition count**.\n\n" +
		"This resource allows you to modify the partition count of an existing Kafka topic " +
		"in your Streamkap cluster. Use this to scale topic throughput.\n\n" +
		"**Note:** Partition count can only be increased, not decreased.\n\n" +
		"[Documentation](https://docs.streamkap.com/topics)",
	Attributes: map[string]schema.Attribute{
		"topic_id": schema.StringAttribute{
			Description:         "The Kafka topic identifier. Format: <source-id>.<schema>.<table> for CDC topics.",
			MarkdownDescription: "The Kafka topic identifier. Format: `<source-id>.<schema>.<table>` for CDC topics.",
			Required:            true,
		},
		"partition_count": schema.Int64Attribute{
			Required:            true,
			Description:         "Number of partitions for the topic. Can only be increased, not decreased. Higher values allow more parallel consumers.",
			MarkdownDescription: "Number of partitions for the topic. Can only be increased, not decreased. Higher values allow more parallel consumers.",
		},
	},
}
```

**Step 2: Verify compilation and run tests**

Run: `go build ./... && go test ./internal/resource/topic/... -v`
Expected: BUILD SUCCESS, PASS

**Step 3: Commit**

```bash
git add internal/resource/topic/topic.go
git commit -m "feat(topic): enhance schema descriptions for AI readability

- Add both Description and MarkdownDescription to all attributes
- Explain topic ID format and partition behavior
- Add resource-level documentation link"
```

---

## Task 10: Update Tag Data Source with Enhanced Descriptions

**Files:**
- Modify: `internal/datasource/tag.go`

**Context:** The tag data source has a schema-level MarkdownDescription but no Description. Also update attribute descriptions.

**Step 1: Update the Schema function**

Add `Description` at schema level and enhance attribute descriptions:

```go
resp.Schema = schema.Schema{
	Description:         "Retrieves information about a Streamkap tag by ID.",
	MarkdownDescription: "Retrieves information about a **Streamkap tag** by ID.\n\n" +
		"Use this data source to look up tag details for use in other resources.\n\n" +
		"[Documentation](https://docs.streamkap.com/tags)",
	Attributes: map[string]schema.Attribute{
		// ... all attributes with enhanced descriptions
	},
}
```

**Step 2: Verify compilation and run tests**

Run: `go build ./... && go test ./internal/datasource/... -v`
Expected: BUILD SUCCESS, PASS

**Step 3: Commit**

```bash
git add internal/datasource/tag.go
git commit -m "feat(datasource/tag): enhance schema descriptions for AI readability

- Add Description field alongside MarkdownDescription
- Enhance attribute descriptions with examples"
```

---

## Task 11: Update Transform Data Source with Enhanced Descriptions

**Files:**
- Modify: `internal/datasource/transform.go`

**Context:** The transform data source has a typo "Tranform" and needs enhanced descriptions.

**Step 1: Fix typo and update the Schema function**

Change `MarkdownDescription: "Tranform data source"` to proper descriptions:

```go
resp.Schema = schema.Schema{
	Description:         "Retrieves information about a Streamkap transform by ID.",
	MarkdownDescription: "Retrieves information about a **Streamkap transform** by ID.\n\n" +
		"Use this data source to look up transform details including topic mappings.\n\n" +
		"[Documentation](https://docs.streamkap.com/transforms)",
	// ... enhanced attributes
}
```

Also add Description/MarkdownDescription to nested topic_map attributes (id, name).

**Step 2: Verify compilation and run tests**

Run: `go build ./... && go test ./internal/datasource/... -v`
Expected: BUILD SUCCESS, PASS

**Step 3: Commit**

```bash
git add internal/datasource/transform.go
git commit -m "feat(datasource/transform): enhance schema descriptions and fix typo

- Fix typo 'Tranform' -> 'Transform'
- Add Description field alongside MarkdownDescription
- Add descriptions to nested attributes"
```

---

## Task 12: Create Basic Example for PostgreSQL Source

**Files:**
- Create: `examples/resources/streamkap_source_postgresql/basic.tf`
- Rename: `examples/resources/streamkap_source_postgresql/resource.tf` → `complete.tf`

**Step 1: Rename existing resource.tf to complete.tf**

Run: `mv examples/resources/streamkap_source_postgresql/resource.tf examples/resources/streamkap_source_postgresql/complete.tf`

**Step 2: Create basic.tf with minimal configuration**

Create `examples/resources/streamkap_source_postgresql/basic.tf` with only required fields and helpful comments.

**Step 3: Add comments to complete.tf**

Add explanatory comments to complete.tf for each field.

**Step 4: Commit**

```bash
git add examples/resources/streamkap_source_postgresql/
git commit -m "docs(examples): add basic.tf for PostgreSQL source

- basic.tf shows minimal required configuration
- Renamed resource.tf to complete.tf for full options
- Added descriptive comments for AI agents"
```

---

## Task 13: Create Basic Examples for Remaining Source Connectors

**Files:**
- Rename and create for: `streamkap_source_mysql`, `streamkap_source_mongodb`, `streamkap_source_dynamodb`, `streamkap_source_sqlserver`, `streamkap_source_kafkadirect`

**Step 1: For each source connector**

1. Rename `resource.tf` to `complete.tf`
2. Create `basic.tf` with minimal required configuration

**Step 2: Commit**

```bash
git add examples/resources/streamkap_source_*/
git commit -m "docs(examples): add basic.tf for all source connectors

- Each source now has basic.tf (minimal) and complete.tf (all options)
- Consistent structure for AI agent reference"
```

---

## Task 14: Create Basic Examples for Destination Connectors

**Files:**
- Rename and create for: `streamkap_destination_snowflake`, `streamkap_destination_clickhouse`, `streamkap_destination_databricks`, `streamkap_destination_postgresql`, `streamkap_destination_s3`, `streamkap_destination_iceberg`, `streamkap_destination_kafka`

**Step 1: For each destination connector**

1. Rename `resource.tf` to `complete.tf`
2. Create `basic.tf` with minimal required configuration

**Step 2: Commit**

```bash
git add examples/resources/streamkap_destination_*/
git commit -m "docs(examples): add basic.tf for all destination connectors

- Each destination now has basic.tf (minimal) and complete.tf (all options)
- Consistent structure for AI agent reference"
```

---

## Task 15: Create Examples for Transform Resources

**Files:**
- Create: `examples/resources/streamkap_transform_map_filter/basic.tf`, `complete.tf`
- Create: `examples/resources/streamkap_transform_enrich/basic.tf`, `complete.tf`
- Create: `examples/resources/streamkap_transform_enrich_async/basic.tf`, `complete.tf`
- Create: `examples/resources/streamkap_transform_sql_join/basic.tf`, `complete.tf`
- Create: `examples/resources/streamkap_transform_rollup/basic.tf`, `complete.tf`
- Create: `examples/resources/streamkap_transform_fan_out/basic.tf`, `complete.tf`

**Context:** Transform resources have NO examples currently. Create basic.tf and complete.tf for each.

**Step 1: Create example directories and files**

For each transform type:
1. Create the directory
2. Create `basic.tf` with minimal configuration
3. Create `complete.tf` with all options

Reference the generated schema files in `internal/generated/transform_*.go` for required fields.

**Step 2: Commit**

```bash
git add examples/resources/streamkap_transform_*/
git commit -m "docs(examples): add examples for all transform resources

- map_filter, enrich, enrich_async, sql_join, rollup, fan_out
- Each has basic.tf (minimal) and complete.tf (all options)"
```

---

## Task 16: Create Example for Tag Resource

**Files:**
- Create: `examples/resources/streamkap_tag/basic.tf`
- Create: `examples/resources/streamkap_tag/complete.tf`

**Context:** Tag resource has NO examples currently.

**Step 1: Create example directory and files**

Create `examples/resources/streamkap_tag/basic.tf`:

```hcl
# Minimal tag configuration

resource "streamkap_tag" "example" {
  name = "my-environment"
  type = ["sources", "destinations", "pipelines"]
}
```

Create `examples/resources/streamkap_tag/complete.tf` with description and all fields.

**Step 2: Commit**

```bash
git add examples/resources/streamkap_tag/
git commit -m "docs(examples): add examples for tag resource

- basic.tf shows minimal configuration
- complete.tf shows all options including description"
```

---

## Task 17: Create Basic Example for Pipeline Resource

**Files:**
- Create: `examples/resources/streamkap_pipeline/basic.tf`
- Rename: `examples/resources/streamkap_pipeline/resource.tf` → `complete.tf`

**Note:** Pipeline has subdirectories (dynamodb_clickhouse, postgresql_clickhouse, sqlserver_databricks). Keep those as additional examples.

**Step 1: Rename resource.tf**

Run: `mv examples/resources/streamkap_pipeline/resource.tf examples/resources/streamkap_pipeline/complete.tf`

**Step 2: Create basic.tf**

Create a minimal pipeline example without transforms.

**Step 3: Commit**

```bash
git add examples/resources/streamkap_pipeline/
git commit -m "docs(examples): add basic.tf for pipeline resource

- basic.tf shows minimal pipeline without transforms
- complete.tf demonstrates all options including transforms and tags
- Keep subdirectory examples for specific use cases"
```

---

## Task 18: Regenerate Documentation

**Files:**
- Regenerate: `docs/resources/*.md`, `docs/data-sources/*.md`

**Step 1: Run documentation generator**

Run: `go generate ./...`
Expected: Documentation files updated in `docs/`

**Step 2: Verify documentation**

Check that:
- Resource descriptions are comprehensive
- All attributes have descriptions
- Examples section includes both basic and complete

**Step 3: Commit**

```bash
git add docs/
git commit -m "docs: regenerate documentation with enhanced descriptions

- All resources have comprehensive descriptions
- Examples include basic and complete configurations
- Ready for Terraform Registry and AI tools"
```

---

## Task 19: Validate AI-Friendliness

**Files:**
- None (validation only)

**Step 1: Build and test**

Run: `go build ./... && go test ./...`
Expected: BUILD SUCCESS, ALL TESTS PASS

**Step 2: Generate fresh documentation**

Run: `go generate ./...`
Expected: SUCCESS

**Step 3: Manual review checklist**

Verify:
- [ ] Every resource has Description and MarkdownDescription
- [ ] Every attribute has Description and MarkdownDescription
- [ ] Enum fields list valid values
- [ ] Fields with defaults mention the default value
- [ ] Sensitive fields have security notes
- [ ] Examples exist for all resources (basic + complete)
- [ ] Data sources have enhanced descriptions
- [ ] Transform examples exist for all 6 types

**Step 4: Final commit (if any fixes needed)**

```bash
git add .
git commit -m "chore: final AI-agent readiness validation

- All checklists verified
- Provider ready for Terraform MCP Server integration"
```

---

## Task 20: Update CHANGELOG.md

**Files:**
- Modify: `CHANGELOG.md`

**Step 1: Add changelog entry**

Add a new section at the top of `CHANGELOG.md`:

```markdown
## [Unreleased]

### Added
- Enhanced schema descriptions for AI-agent compatibility (Terraform MCP Server)
  - All resources now have both `Description` (plain text) and `MarkdownDescription` (rich text)
  - Enum fields list all valid values in descriptions
  - Default values are documented in field descriptions
  - Sensitive fields include security notes
- Basic and complete example configurations for all resources
- Examples for all 6 transform types (previously missing)
- Example for tag resource (previously missing)
- Resource-level documentation with links to Streamkap docs

### Fixed
- Fixed typo "Tranform" in transform data source

### Changed
- Regenerated all connector schemas with comprehensive descriptions
- Restructured examples directory with `basic.tf` and `complete.tf` patterns
```

**Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): document AI-agent readiness improvements"
```

---

## Task 21: Update README.md with AI-Agent Section

**Files:**
- Modify: `README.md`

**Step 1: Add AI-Agent Compatibility section**

Add after the "Usage" section explaining MCP Server integration and AI-friendly features.

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs(readme): add AI-agent compatibility section

- Explain MCP Server integration
- Document AI-friendly features
- Provide usage examples for AI assistants"
```

---

## Task 22: Create AI-Agent Documentation Page

**Files:**
- Create: `docs/AI_AGENT_COMPATIBILITY.md`

**Step 1: Create comprehensive documentation**

Create dedicated documentation explaining:
- How the Terraform MCP Server works
- What schema fields AI agents read
- Example structure (basic.tf/complete.tf)
- Testing AI compatibility
- Contributing guidelines for future resources

**Step 2: Commit**

```bash
git add docs/AI_AGENT_COMPATIBILITY.md
git commit -m "docs: add AI-agent compatibility guide

- Explain MCP Server integration
- Document schema description standards
- Provide testing guidance"
```

---

## Task 23: Update CLAUDE.md with AI Description Standards

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Add AI Description Standards section**

Add a section documenting the standards for future development:
- Schema-level description patterns
- Attribute description patterns
- Enum field requirements
- Default value documentation
- Sensitive field security notes
- Example file structure

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs(claude): add AI description standards

- Document schema description patterns
- Ensure future resources are AI-agent ready"
```

---

## Summary

This plan transforms the Terraform provider to be AI-agent ready by:

1. **Tasks 1-5**: Enhance the `tfgen` code generator to produce rich descriptions (MarkdownDescription field, valid values, defaults, security notes)
2. **Task 6**: Regenerate all connector schemas with new descriptions
3. **Tasks 7-9**: Manually update hand-coded resources (pipeline, tag, topic)
4. **Tasks 10-11**: Update data sources (tag, transform) with enhanced descriptions
5. **Tasks 12-17**: Create/restructure examples (all sources, destinations, transforms, tag, pipeline)
6. **Tasks 18-19**: Regenerate documentation and validate
7. **Tasks 20-23**: Update documentation (CHANGELOG, README, AI guide, CLAUDE.md)

Total: 23 tasks, each with discrete, testable steps.
