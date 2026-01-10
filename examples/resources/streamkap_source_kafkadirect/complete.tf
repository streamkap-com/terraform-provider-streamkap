# Complete Kafka Direct source configuration
# This example shows all available configuration options for reading data
# directly from external Kafka topics

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
  # Display name for this source in Streamkap UI
  name = "example-source-kafkadirect"

  # Topic configuration
  topic_prefix       = "sample-topic_"                                              # Prefix for topic names
  topic_include_list = "sample-topic_topic1,sample-topic_topic2,sample-topic_topic3" # Topics to sync (comma-separated)

  # Data format configuration
  format = "json" # Options: json, string

  # Schema handling
  # If false (default), Streamkap infers schema from data
  # If true, messages must contain 'schema' and 'payload' structures
  schemas_enable = true
}

output "example-source-kafkadirect" {
  value = streamkap_source_kafkadirect.example-source-kafkadirect.id
}
