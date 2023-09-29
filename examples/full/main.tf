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
  name      = "My Stream"
  connector = "postgresql"

  config = jsonencode({
    "database.hostname.user.defined"            = var.source_host
    "database.port"                             = "5432"
    "database.user"                             = "tf"
    "database.password"                         = var.source_password
    "database.dbname"                           = "tf"
    "schema.include.list"                       = "public"
    "table.include.list.user.defined"           = "tf.test"
    "signal.data.collection.schema.or.database" = "tf"
    "slot.name"                                 = "streamkap_pgoutput_slot"
    "publication.name"                          = "streamkap_pub"
    "database.sslmode"                          = "require"
    "binary.handling.mode"                      = "bytes"
    "snapshot.mode.user.defined"                = "Initial"
    "incremental.snapshot.chunk.size"           = 1024
    "max.batch.size"                            = 2048
  })
}

resource "streamkap_destination" "postgresql" {
  name      = "My Stream"
  connector = "postgresql"

  config = jsonencode({
    "database.hostname.user.defined"            = var.destination_host
    "database.port"                             = "5432"
    "database.user"                             = "tf"
    "database.password"                         = var.destination_password
    "database.dbname"                           = "tf"
    "schema.include.list"                       = "public"
    "table.include.list.user.defined"           = "tf.test"
    "signal.data.collection.schema.or.database" = "tf"
    "slot.name"                                 = "streamkap_pgoutput_slot"
    "publication.name"                          = "streamkap_pub"
    "database.sslmode"                          = "require"
    "binary.handling.mode"                      = "bytes"
    "snapshot.mode.user.defined"                = "Initial"
    "incremental.snapshot.chunk.size"           = 1024
    "max.batch.size"                            = 2048
  })
}

resource "streamkap_pipeline" "this" {
  name = "My Stream"

  source = {
    id = {
      "$oid" : streamkap_source.postgresql.id
    }
    name      = streamkap_source.postgresql.name
    connector = streamkap_source.postgresql.connector
    topics    = ["test1"]
  }

  destination = {
    id = {
      "$oid" : streamkap_destination.postgresql.id
    }
    name      = streamkap_destination.postgresql.name
    connector = streamkap_destination.postgresql.connector
  }
}