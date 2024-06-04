terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
  required_version = ">= 1.0"
}

provider "streamkap" {}

resource "streamkap_destination_snowflake" "example-destination-snowflake" {
  name                             = "test-destination-snowflake"
  snowflake_url_name               = ""
  snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
  snowflake_private_key            = ""
  snowflake_private_key_passphrase = ""
  snowflake_database_name          = "STREAMKAP_POSTGRESQL"
  snowflake_schema_name            = "STREAMKAP"
  snowflake_role_name              = "STREAMKAP_ROLE"
}

output "example-destination-snowflake" {
  value = streamkap_destination_snowflake.example-destination-snowflake.id
}