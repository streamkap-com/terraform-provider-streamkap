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

resource "streamkap_source_kafkadirect" "example-source-kafkadirect" {
  name               = "test-source-kafkadirect"
  topic_prefix       = "sample-topic_"
  kafka_format       = "json"
  schemas_enable     = true
  topic_include_list = "sample-topic_topic1, sample-topic_topic2, sample-topic_topic3"
}

output "example-source-kafkadirect" {
  value = streamkap_source_kafkadirect.example-source-kafkadirect.id
}
