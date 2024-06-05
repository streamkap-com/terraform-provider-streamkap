terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
    }
  }
  required_version = ">= 1.0"
}

provider "streamkap" {}

variable "destination_snowflake_url_name" {
  type        = string
  description = "The URL name of the Snowflake database"
}

variable "destination_snowflake_private_key" {
  type        = string
  sensitive   = true
  description = "The private key of the Snowflake database"
}

variable "destination_snowflake_key_passphrase" {
  type        = string
  sensitive   = true
  description = "The passphrase of the private key of the Snowflake database"
}

resource "streamkap_destination_snowflake" "example-destination-snowflake" {
  name                             = "test-destination-snowflake"
  snowflake_url_name               = var.destination_snowflake_url_name
  snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
  snowflake_private_key            = var.destination_snowflake_private_key
  snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
  snowflake_database_name          = "STREAMKAP_POSTGRESQL"
  snowflake_schema_name            = "STREAMKAP"
  snowflake_role_name              = "STREAMKAP_ROLE"
}

output "example-destination-snowflake" {
  value = streamkap_destination_snowflake.example-destination-snowflake.id
}