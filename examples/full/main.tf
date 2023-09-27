terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
}

provider "streamkap" {
  host      = "https://aead568tp7.execute-api.us-west-2.amazonaws.com/dev"
  client_id = var.client_id
  secret    = var.secret_key
}

data "streamkap_token" "this" {}

resource "streamkap_source" "mysql" {
  name      = "My Stream"
  connector = "mysql"

  config = jsonencode({
    "database.hostname.user.defined"            = "192.168.3.47"
    "database.port"                             = "3306"
    "database.user"                             = "root"
    "database.password"                         = "iAxki9j9fr8H8LV"
    "database.include.list.user.defined"        = "database1, database2"
    "table.include.list.user.defined"           = "database1.table1, database1.table2, database2.table3, database2.table4"
    "signal.data.collection.schema.or.database" = "test1"
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
  connector = "postgresql"

  config = jsonencode({
    "database.hostname.user.defined"            = "192.168.3.47"
    "database.port"                             = "3306"
    "database.user"                             = "root"
    "database.password"                         = "iAxki9j9fr8H8LV"
    "database.include.list.user.defined"        = "database1, database2"
    "table.include.list.user.defined"           = "database1.table1, database1.table2, database2.table3, database2.table4"
    "signal.data.collection.schema.or.database" = "test1"
    "database.connectionTimeZone"               = "SERVER"
    "snapshot.gtid"                             = "No"
    "snapshot.mode.user.defined"                = "When Needed"
    "binary.handling.mode"                      = "bytes"
    "incremental.snapshot.chunk.size"           = 1024
    "max.batch.size"                            = 2048
  })
}

resource "streamkap_pipeline" "this" {
  name = "My Stream"

  source = {
    id = {
      "$oid" : streamkap_source.mysql.id
    }
    name      = streamkap_source.mysql.name
    connector = streamkap_source.mysql.connector
    topics    = ["test1"]
  }

  destination = {
    id = {
      "$oid" : streamkap_destination.mysql.id
    }
    name      = streamkap_destination.mysql.name
    connector = streamkap_destination.mysql.connector
  }
}