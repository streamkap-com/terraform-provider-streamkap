terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      # Set version to the release matching this documentation.
      # Beta releases require an exact prerelease version; omission selects stable.
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

resource "streamkap_destination_kafka" "example-destination-kafka" {
  name                 = "example-destination-kafka"
  kafka_sink_bootstrap = "kafka.example.com:9098"
  destination_format   = "avro"
  schema_registry_url  = "https://schema-registry.example.com"
}

output "example-destination-kafka" {
  value = streamkap_destination_kafka.example-destination-kafka.id
}
