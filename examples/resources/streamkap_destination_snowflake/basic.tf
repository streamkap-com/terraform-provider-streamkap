# Minimal Snowflake destination configuration

resource "streamkap_destination_snowflake" "example" {
  name = "my-snowflake-dest"

  snowflake_url_name      = var.snowflake_url
  snowflake_user_name     = var.snowflake_user
  snowflake_private_key   = var.snowflake_private_key
  snowflake_database_name = "ANALYTICS"
  snowflake_schema_name   = "PUBLIC"
}

variable "snowflake_url" {
  description = "Snowflake account URL (e.g., account.snowflakecomputing.com)"
  type        = string
}

variable "snowflake_user" {
  description = "Snowflake username"
  type        = string
}

variable "snowflake_private_key" {
  description = "Snowflake private key for authentication"
  type        = string
  sensitive   = true
}
