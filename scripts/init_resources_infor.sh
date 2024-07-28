#! /bin/bash
# Sources----------------------------------------------------
# PostgreSQL
export TF_VAR_source_postgresql_hostname=""
export TF_VAR_source_postgresql_password=''

# DynamoDB
export TF_VAR_source_dynamodb_aws_region=""
export TF_VAR_source_dynamodb_aws_access_key_id=""
export TF_VAR_source_dynamodb_aws_secret_key=""

# Destinations-----------------------------------------------
# Snowflake
export TF_VAR_destination_snowflake_url_name=""
export TF_VAR_destination_snowflake_private_key=''
export TF_VAR_destination_snowflake_key_passphrase=""

# ClickHouse
export TF_VAR_destination_clickhouse_hostname=""
export TF_VAR_destination_clickhouse_connection_username=""
export TF_VAR_destination_clickhouse_connection_password=''