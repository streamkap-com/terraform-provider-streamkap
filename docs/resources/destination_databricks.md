---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "streamkap_destination_databricks Resource - terraform-provider-streamkap"
subcategory: ""
description: |-
  Destination Databricks resource
---

# streamkap_destination_databricks (Resource)

Destination Databricks resource

## Example Usage

```terraform
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "destination_databricks_connection_url" {
  type        = string
  description = "The JDBC url of the databricks server"
}

variable "destination_databricks_token" {
  type        = string
  sensitive   = true
  description = "The token for the Databricks database"
}

resource "streamkap_destination_databricks" "example-destination-databricks" {
  name              = "example-destination-databricks"
  table_name_prefix = "streamkap"
  ingestion_mode    = "append"
  partition_mode    = "by_topic"
  hard_delete       = true
  tasks_max         = 5
  connection_url    = var.destination_databricks_connection_url
  databricks_token  = var.destination_databricks_token
  schema_evolution  = "basic"
}

output "example-destination-databricks" {
  value = streamkap_destination_databricks.example-destination-databricks.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `connection_url` (String) JDBC URL
- `databricks_token` (String, Sensitive) Token
- `name` (String) Destination name
- `table_name_prefix` (String) Schema for the associated table name

### Optional

- `hard_delete` (Boolean) Specifies whether the connector processes DELETE or tombstone events and removes the corresponding row from the database (applies to `upsert` only)
- `ingestion_mode` (String) `upsert` or `append` modes are available
- `partition_mode` (String) Partition tables or not
- `schema_evolution` (String) Controls how schema evolution is handled by the sink connector. For pipelines with pre-created destination tables, set to `none`
- `tasks_max` (Number) The maximum number of active task

### Read-Only

- `connector` (String)
- `id` (String) Destination Databricks identifier

## Import

Import is supported using the following syntax:

```shell
# Destination Snowflake can be imported by specifying the identifier.
terraform import streamkap_destination_databricks.example-destination-databricks 665e894ebb3753f38d983cee
```
