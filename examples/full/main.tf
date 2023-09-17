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

resource "streamkap_source" "this" {
  name      = "My Stream"
  connector = "mysql"
  config = {
    "database.hostname.user.defined" = "192.168.3.47"
    "database.port" = "3306"
    "database.user" =  "root"
    "database.password" = "iAxki9j9fr8H8LV"
    "database.include.list.user.defined" = "database1, database2"
    "table.include.list.user.defined" = "database1.table1, database1.table2, database2.table3, database2.table4"
    "signal.data.collection.schema.or.database" = "test1"
    "database.connectionTimeZone" = "SERVER"
    "snapshot.gtid" = "No"
    "snapshot.mode.user.defined" = "When Needed"
    "binary.handling.mode" = "bytes"
    "incremental.snapshot.chunk.size" = 1024
    "max.batch.size" = 2048
  }
}

#resource "streamkap_destination" "this" {
#  name      = "My Stream"
#  connector = "postgresql"
#}
#
#resource "streamkap_pipeline" "this" {
#  name      = "My Stream"
#  sub_id    = "sub-id-123"
#  tenant_id = "tenant-id-123"
#}