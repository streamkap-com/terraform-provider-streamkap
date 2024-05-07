terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
}
provider "streamkap" {
  host      = var.host
  client_id = var.client_id
  secret    = var.secret_key
}
data "streamkap_token" "this" {}
resource "streamkap_source" "postgresql" {
  name      = "My Postgres" # The name of the source
  connector = "postgresql"  # Please do not change this
  config = jsonencode({
    "database.hostname.user.defined"            = var.source_host
    "database.port"                             = "5432"
    "database.user"                             = "postgres"
    "database.password"                         = var.source_password
    "signal.data.collection.schema.or.database" = "public"
    "database.dbname"                           = "my_database"
    "schema.include.list"                       = "schema1, schema2"
    "table.include.list.user.defined"           = "schema1.table1, schema2.table2, schema2.table3"
    "slot.name"                                 = "streamkap_pgoutput_slot"
    "publication.name"                          = "streamkap_pub"
    "database.sslmode"                          = "require"
    "snapshot.max.threads"                      = "1"
    "snapshot.fetch.size"                       = "102400"
    "snapshot.mode.user.defined"                = "Initial"
    "binary.handling.mode"                      = "bytes"
    "incremental.snapshot.chunk.size"           = 102400
    "max.batch.size"                            = 2048
    "max.queue.size.user.defined"               = "204800"
  })
}
resource "streamkap_destination" "snowflake" {
  name      = "My Snowflake" # The name of the destination
  connector = "snowflake"    # Please do not change this
  config = jsonencode({
    "snowflake.url.name"               = var.destination_host
    "tasks.max"                        = 1
    "snowflake.user.name"              = "root"
    "snowflake.private.key"            = var.destination_private_key
    "snowflake.private.key.passphrase" = var.destination_key_passphrase
    "snowflake.database.name"          = "databasename"
    "snowflake.schema.name"            = "schemaname"
    "snowflake.role.name"              = "STREAMKAP_ROLE"
  })
}
resource "streamkap_pipeline" "this" {
  name = "My Pipeline" # The name of the pipeline
  source = {
    id =  streamkap_source.postgresql.id # The id of the source
    name      = streamkap_source.postgresql.name                   # The name of the source
    connector = streamkap_source.postgresql.connector              # The connector of the source
    topics    = ["schema1.table1, schema2.table2, schema2.table3"] # List of tables of the source that we want to sync to destination
  }
  destination = {
    id = streamkap_destination.snowflake.id # The id of the destination
    name      = streamkap_destination.snowflake.name      # The name of the destination
    connector = streamkap_destination.snowflake.connector # The connector of the destination
  }
}
