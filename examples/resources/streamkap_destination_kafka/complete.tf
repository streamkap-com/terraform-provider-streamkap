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

resource "streamkap_destination_kafka" "example-destination-kafka" {
  name                 = "example-destination-kafka"
  kafka_sink_bootstrap = "kafka-controller.streamkap.net:9098"
  destination_format   = "avro"
  schema_registry_url  = "https://schema-registry-dev.streamkap.net"
}

output "example-destination-kafka" {
  value = streamkap_destination_kafka.example-destination-kafka.id
}
