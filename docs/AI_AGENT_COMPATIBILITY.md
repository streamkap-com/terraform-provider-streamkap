# AI-Agent Compatibility Guide

This document explains how the Streamkap Terraform provider is optimized for AI assistants and the Terraform MCP Server.

## Overview

The Streamkap provider implements AI-agent readiness patterns that enable AI coding assistants to:
- Generate accurate Terraform configurations
- Understand valid values for enum fields
- Know default values without reading documentation
- Identify sensitive fields requiring secure handling

## Terraform MCP Server Integration

The [Terraform MCP Server](https://github.com/hashicorp/terraform-mcp-server) exposes provider schemas to AI assistants. When an AI queries the Streamkap provider schema, it receives:

1. **Schema-level descriptions** explaining what each resource does
2. **Attribute-level descriptions** with:
   - Purpose of the field
   - Valid values for enums (e.g., "Valid values: `insert`, `upsert`")
   - Default values (e.g., "Defaults to `5432`")
   - Security notes for sensitive fields

## Schema Description Patterns

### Resource-Level Descriptions

Each resource has both `Description` (plain text for CLI) and `MarkdownDescription` (rich text):

```go
Description: "Manages a PostgreSQL source connector.",
MarkdownDescription: "Manages a **PostgreSQL source connector**.\n\n" +
    "This resource creates and manages a PostgreSQL source for Streamkap data pipelines.\n\n" +
    "[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",
```

### Enum Field Descriptions

Enum fields explicitly list all valid values:

```
Description: "Compression type for files. Defaults to `gzip`. Valid values: `none`, `gzip`, `snappy`, `zstd`."
```

### Default Value Documentation

Fields with defaults document them inline:

```
Description: "PostgreSQL Port. Defaults to `5432`."
```

### Sensitive Field Security Notes

Sensitive fields include security warnings:

```
MarkdownDescription: "The database password.\n\n**Security:** This value is marked sensitive and will not appear in CLI output or logs."
```

## Example Structure

Each resource has two example files:

### `basic.tf` - Minimal Configuration

Shows only required fields with helpful comments:

```hcl
# Minimal PostgreSQL source configuration

variable "db_password" {
  type      = string
  sensitive = true
}

resource "streamkap_source_postgresql" "example" {
  name              = "my-postgresql-source"
  database_hostname = "db.example.com"
  database_user     = "streamkap"
  database_password = var.db_password
  database_dbname   = "mydb"
  table_include_list = "public.customers,public.orders"
}
```

### `complete.tf` - All Options

Shows all available attributes with comments explaining each:

```hcl
# Complete PostgreSQL source with all options

terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
}

resource "streamkap_source_postgresql" "example" {
  name = "example-source-postgresql"

  # Connection settings
  database_hostname = var.db_host
  database_port     = "5432"  # Default
  database_user     = "streamkap"
  database_password = var.db_password
  database_dbname   = "mydb"

  # SSL configuration - options: require, disable
  database_sslmode = "require"

  # Table selection
  table_include_list = "public.customers,public.orders"

  # ... additional options
}
```

## AI Assistant Usage Patterns

### Generating New Resources

AI assistants can use the schema to generate accurate configurations:

**Prompt:**
```
Create a Streamkap pipeline from PostgreSQL to Snowflake with upsert mode enabled.
```

**AI understands from schema:**
- Required fields for `streamkap_source_postgresql`
- Required fields for `streamkap_destination_snowflake`
- `insert_mode` accepts "insert" or "upsert"
- Sensitive fields need variable references

### Debugging Configurations

AI can identify issues by checking:
- Required vs optional fields
- Valid enum values
- Type constraints (string, int, bool)
- Default values

### Security Recommendations

AI can recommend best practices:
- Use variables with `sensitive = true` for passwords/keys
- Don't hardcode credentials
- Use SSH tunnels for private networks

## Contributing

When adding new resources or modifying schemas:

1. **Always include both `Description` and `MarkdownDescription`**
2. **Document enum values**: Add "Valid values: X, Y, Z" to descriptions
3. **Document defaults**: Add "Defaults to X" to descriptions
4. **Mark sensitive fields**: Set `Sensitive: true` and add security note
5. **Create examples**: Add both `basic.tf` and `complete.tf`

### tfgen Code Generator

The `cmd/tfgen` tool automatically generates schemas with these patterns from backend configuration files. See the generator for implementation details.

## Testing AI Compatibility

To verify AI-agent readiness:

1. **Schema inspection**: Check that `MarkdownDescription` fields are populated
2. **Enum coverage**: Verify all enum fields list valid values
3. **Default documentation**: Confirm defaults are documented
4. **Sensitive marking**: Ensure sensitive fields have security notes
5. **Example completeness**: Verify examples demonstrate all patterns

## Resources

- [Terraform MCP Server](https://github.com/hashicorp/terraform-mcp-server)
- [Terraform Plugin Framework - Schema](https://developer.hashicorp.com/terraform/plugin/framework/schemas)
- [Streamkap Documentation](https://docs.streamkap.com)
