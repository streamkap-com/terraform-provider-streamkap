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

variable "destination_snowflake_url_name" {
  type        = string
  description = "The URL name of the Snowflake database"
}
variable "destination_snowflake_private_key" {
  type        = string
  sensitive   = true
  description = "Snowflake private key without PEM headers, footers, or line breaks."
}
variable "destination_snowflake_key_passphrase" {
  type        = string
  sensitive   = true
  description = "The passphrase of the private key of the Snowflake database"
}
resource "streamkap_destination_snowflake" "example-destination-snowflake" {
  name                             = "example-destination-snowflake"
  snowflake_url_name               = var.destination_snowflake_url_name
  snowflake_user_name              = "STREAMKAP_USER"
  snowflake_private_key            = var.destination_snowflake_private_key
  snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
  sfwarehouse                      = "STREAMKAP_WH"
  snowflake_database_name          = "STREAMKAP"
  snowflake_schema_name            = "PUBLIC"
  snowflake_role_name              = "STREAMKAP_ROLE"
  ingestion_mode                   = "upsert"
  hard_delete                      = true
  use_hybrid_tables                = false
  apply_dynamic_table_script       = false
  snowflake_topic2table_map        = "REGEX_MATCHER>^([-\\w]+\\.)([-\\w]+\\.)?([-\\w]+\\.)?([-\\w]+\\.)?([-\\w]+):$5"

}

output "example-destination-snowflake" {
  value = streamkap_destination_snowflake.example-destination-snowflake.id
}
