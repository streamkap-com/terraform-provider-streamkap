---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "streamkap_source_postgresql Resource - terraform-provider-streamkap"
subcategory: ""
description: |-
  Source PostgreSQL resource
---

# streamkap_source_postgresql (Resource)

Source PostgreSQL resource



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `database_dbname` (String) Database from which to stream data
- `database_hostname` (String) PostgreSQL Hostname. For example, postgres.something.rds.amazonaws.com
- `database_password` (String, Sensitive) Password to access the database
- `database_user` (String) Username to access the database
- `name` (String) Source name
- `schema_include_list` (String) Schemas to include
- `table_include_list` (String) Source tables to sync

### Optional

- `binary_handling_mode` (String) Representation of binary data for binary columns
- `database_port` (Number) PostgreSQL Port. For example, 5432
- `database_sslmode` (String) Whether to use an encrypted connection to the PostgreSQL server
- `heartbeat_data_collection_schema_or_database` (String) Schema for heartbeat data collection
- `heartbeat_enabled` (Boolean) Enable heartbeat to keep the pipeline healthy during low data volume
- `include_source_db_name_in_table_name` (Boolean) Prefix topics with the database name
- `publication_name` (String) Publication name for the connector
- `signal_data_collection_schema_or_database` (String) Schema for signal data collection
- `slot_name` (String) Replication slot name for the connector
- `ssh_enabled` (Boolean) Connect via SSH tunnel
- `ssh_host` (String) Hostname of the SSH server, only required if `ssh_enabled` is true
- `ssh_port` (String) Port of the SSH server, only required if `ssh_enabled` is true
- `ssh_user` (String) User for connecting to the SSH server, only required if `ssh_enabled` is true

### Read-Only

- `connector` (String)
- `id` (String) Source PostgreSQL identifier