terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "~> 2.2"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

# PostgreSQL and Snowflake must already be configured for Streamkap access.
variable "postgresql_hostname" {
  type = string
}

variable "postgresql_password" {
  type      = string
  sensitive = true
}

variable "snowflake_url" {
  type        = string
  description = "Snowflake account URL, such as org-account.snowflakecomputing.com."
}

variable "snowflake_private_key" {
  type        = string
  sensitive   = true
  description = "Snowflake private key without PEM headers, footers, or line breaks."
}

resource "streamkap_source_postgresql" "example" {
  name                = "orders-postgres"
  database_hostname   = var.postgresql_hostname
  database_port       = 5432
  database_user       = "streamkap"
  database_password   = var.postgresql_password
  database_dbname     = "commerce"
  database_sslmode    = "require"
  schema_include_list = "public"
  table_include_list  = "public.orders,public.customers"
}

resource "streamkap_destination_snowflake" "example" {
  name                    = "analytics-snowflake"
  snowflake_url_name      = var.snowflake_url
  snowflake_user_name     = "STREAMKAP_USER"
  snowflake_private_key   = var.snowflake_private_key
  snowflake_database_name = "STREAMKAP"
  snowflake_schema_name   = "PUBLIC"
  snowflake_role_name     = "STREAMKAP_ROLE"
  sfwarehouse             = "STREAMKAP_WH"
  ingestion_mode          = "upsert"
  hard_delete             = true
}

resource "streamkap_pipeline" "example" {
  name                = "orders-to-snowflake"
  snapshot_new_tables = true

  source = {
    id        = streamkap_source_postgresql.example.id
    name      = streamkap_source_postgresql.example.name
    connector = streamkap_source_postgresql.example.connector
    topics    = ["public.orders", "public.customers"]
  }

  destination = {
    id        = streamkap_destination_snowflake.example.id
    name      = streamkap_destination_snowflake.example.name
    connector = streamkap_destination_snowflake.example.connector
  }

}

output "pipeline_id" {
  value = streamkap_pipeline.example.id
}
