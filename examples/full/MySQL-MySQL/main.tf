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
    "database.hostname.user.defined"     = var.source_host
    "database.port"                      = "3306"
    "database.user"                      = "root"
    "database.password"                  = var.source_password
    "database.include.list.user.defined" = "database1, database2"
    "table.include.list.user.defined"    = "database1.table1, database2.table2"
    "database.connectionTimeZone"        = "SERVER"
    "snapshot.gtid"                      = "Yes"
    "snapshot.mode.user.defined"         = "When Needed"
    "binary.handling.mode"               = "bytes"
    "snapshot.max.threads"               = "1"
    "snapshot.fetch.size"                = "102400"
    "incremental.snapshot.chunk.size"    = 102400
    "incremental.snapshot.chunk.size"    = 1024
    "max.batch.size"                     = 2048
  })
}
resource "streamkap_destination" "mysql" {
  name      = "My Stream 1"
  connector = "mysql"
  config = jsonencode({
    "database.hostname.user.defined" = var.destination_host
    "database.port.user.defined"     = "3306"
    "database.database.user.defined" = "test"
    "connection.username"            = "root"
    "connection.password"            = var.destination_password
    "delete.enabled"                 = true
    "insert.mode"                    = "insert"
    "schema.evolution"               = "none"
    "tasks.max"                      = 1
    "primary.key.mode"               = "record_key"
    "primary.key.fields"             = "id"
  })
}
resource "streamkap_pipeline" "this" {
  name = "My Stream 2"
  source = {
    id = {
      "oid" : streamkap_source.mysql.id
    }
    name      = streamkap_source.mysql.name
    connector = streamkap_source.mysql.connector
    topics    = ["database1.table1", "database2.table2"]
  }
  destination = {
    id = {
      "oid" : streamkap_destination.mysql.id
    }
    name      = streamkap_destination.mysql.name
    connector = streamkap_destination.mysql.connector
  }
}
