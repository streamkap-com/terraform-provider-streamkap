terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

resource "streamkap_topic" "example-topic" {
  topic_id        = "source_67adbcc172417ef6338e01a1.default.tst-junit-2"
  partition_count = 25
}

resource "streamkap_topic" "example-topic2" {
  topic_id        = "source_67c70fe9239df43df3617809.test.js_orders1"
  partition_count = 25
}

output "example-topic" {
  value = streamkap_topic.example-topic.topic_id
}