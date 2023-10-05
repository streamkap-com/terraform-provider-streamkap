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
resource "streamkap_source" "mysql" {
  name      = "My Stream"
  connector = "mysql"
  config = jsonencode({
    "database.hostname.user.defined"            = var.source_host
    "database.port"                             = "3306"
    "database.user"                             = "root"
    "database.password"                         = var.source_password
    "database.include.list.user.defined"        = "test"
    "table.include.list.user.defined"           = "test.test"
    "signal.data.collection.schema.or.database" = "test"
    "database.connectionTimeZone"               = "SERVER"
    "snapshot.gtid"                             = "No"
    "snapshot.mode.user.defined"                = "When Needed"
    "binary.handling.mode"                      = "bytes"
    "incremental.snapshot.chunk.size"           = 1024
    "max.batch.size"                            = 2048
  })
}
resource "streamkap_destination" "mysql" {
  name      = "My Stream"
  connector = "mysql"
  config = jsonencode({
    "database.hostname.user.defined" = var.destination_host
    "database.port.user.defined"     = "3306"
    "connection.username"            = "root"
    "connection.password"            = var.destination_password
    "database.database.user.defined" = "test"
    "delete.enabled"                 = true
    "insert.mode"                    = "upsert"
    "schema.evolution"               = "basic"
    "slot.name"                      = "streamkap_pgoutput_slot"
    "tasks.max"                      = 1
    "primary.key.mode"               = "record_key"
    "primary.key.fields"             = "id"
  })
}
resource "streamkap_pipeline" "this" {
  name = "My Stream 1"
  source = {
    id = {
      "oid" : streamkap_source.mysql.id
    }
    name      = streamkap_source.mysql.name
    connector = streamkap_source.mysql.connector
    topics    = ["test.test"]
  }
  destination = {
    id = {
      "oid" : streamkap_destination.mysql.id
    }
    name      = streamkap_destination.mysql.name
    connector = streamkap_destination.mysql.connector
  }
}