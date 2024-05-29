resource "streamkap_destination_snowflake" "test" {
  name                             = "test-destination-snowflake"
  snowflake_url_name               = "%s"
  snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
  snowflake_private_key            = "%s"
  snowflake_private_key_passphrase = "%s"
  snowflake_database_name          = "STREAMKAP_POSTGRESQL"
  snowflake_schema_name            = "STREAMKAP"
  snowflake_role_name              = "STREAMKAP_ROLE"
}