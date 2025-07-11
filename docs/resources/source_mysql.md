---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "streamkap_source_mysql Resource - terraform-provider-streamkap"
subcategory: ""
description: |-
  Source MySQL resource
---

# streamkap_source_mysql (Resource)

Source MySQL resource

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

variable "source_mysql_hostname" {
  type        = string
  description = "The hostname of the MySQL database"
}

variable "source_mysql_password" {
  type        = string
  sensitive   = true
  description = "The password of the MySQL database"
}

resource "streamkap_source_mysql" "test" {
  name                                      = "test-source-mysql"
  database_hostname                         = var.source_mysql_hostname
  database_port                             = 3306
  database_user                             = "admin"
  database_password                         = var.source_mysql_password
  database_include_list                     = "crm,ecommerce,tst"
  table_include_list                        = "crm.demo,ecommerce.customers,tst.test_id_timestamp"
  signal_data_collection_schema_or_database = "crm"
  column_include_list                       = "crm[.]demo[.](id|name),ecommerce[.]customers[.](customer_id|email)"
  database_connection_timezone              = "SERVER"
  snapshot_gtid                             = true
  binary_handling_mode                      = "bytes"
  ssh_enabled                               = false
}

output "example-source-mysql" {
  value = streamkap_source_mysql.example-source-mysql.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `database_hostname` (String) MySQL Hostname. For example, mysqldb.something.rds.amazonaws.com
- `database_include_list` (String) Source Databases
- `database_password` (String, Sensitive) Password to access the database
- `database_user` (String) Username to access the database
- `name` (String) Source name
- `table_include_list` (String) Source tables to sync

### Optional

- `binary_handling_mode` (String) Specifies how the data for binary columns e.g. blob, binary, varbinary should be represented. This setting depends on what the destination is. See the documentation for more details.
- `column_exclude_list` (String) Comma separated list of columns blacklist regular expressions, format schema[.]table[.](column1|column2|etc)
- `column_include_list` (String) Comma separated list of columns whitelist regular expressions, format schema[.]table[.](column1|column2|etc)
- `database_connection_timezone` (String) Set the connection timezone. If set to SERVER, the source will detect the connection time zone from the values configured on the MySQL server session variables 'time_zone' or 'system_time_zone'
- `database_port` (Number) MySQL Port. For example, 3306
- `heartbeat_data_collection_schema_or_database` (String) Heartbeat Table Database
- `heartbeat_enabled` (Boolean) Heartbeats are used to keep the pipeline healthy when there is a low volume of data at times.
- `insert_static_key_field_1` (String) The name of the static field to be added to the message key.
- `insert_static_key_field_2` (String) The name of the static field to be added to the message key.
- `insert_static_key_value_1` (String) The value of the static field to be added to the message key.
- `insert_static_key_value_2` (String) The value of the static field to be added to the message key.
- `insert_static_value_1` (String) The value of the static field to be added to the message value.
- `insert_static_value_2` (String) The value of the static field to be added to the message value.
- `insert_static_value_field_1` (String) The name of the static field to be added to the message value.
- `insert_static_value_field_2` (String) The name of the static field to be added to the message value.
- `predicates_istopictoenrich_pattern` (String) Regex pattern to match topics for enrichment
- `signal_data_collection_schema_or_database` (String) Schema for signal data collection. If connector is in read-only mode (snapshot_gtid="Yes"), set this to null.
- `snapshot_gtid` (Boolean) GTID snapshots are read only but require some prerequisite settings, including enabling GTID on the source database. See the documentation for more details.
- `ssh_enabled` (Boolean) Connect via SSH tunnel
- `ssh_host` (String) Hostname of the SSH server, only required if `ssh_enabled` is true
- `ssh_port` (String) Port of the SSH server, only required if `ssh_enabled` is true
- `ssh_user` (String) User for connecting to the SSH server, only required if `ssh_enabled` is true

### Read-Only

- `connector` (String)
- `id` (String) Source MySQL identifier

## Import

Import is supported using the following syntax:

```shell
# Source MySQL can be imported by specifying the identifier.
terraform import streamkap_source_mysql.example-source-mysql 665e894ebb3753f38d983cee
```
