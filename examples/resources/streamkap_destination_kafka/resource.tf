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

variable "kafka_bootstrap_servers" {
  type        = string
  description = "Comma-separated list of Kafka broker addresses"
  default     = "localhost:9092"
}

# Example with JSON format
resource "streamkap_destination_kafka" "example-destination-kafka-json" {
  name                 = "example-destination-kafka-json"
  kafka_sink_bootstrap = var.kafka_bootstrap_servers
  destination_format   = "json"
  json_schema_enable   = false
  topic_prefix         = "output-"
  topic_suffix         = "-processed"
}

# Example with Avro format
resource "streamkap_destination_kafka" "example-destination-kafka-avro" {
  name                             = "example-destination-kafka-avro"
  kafka_sink_bootstrap             = var.kafka_bootstrap_servers
  destination_format               = "avro"
  schema_registry_url_user_defined = "http://schema-registry:8081"
  topic_prefix                     = "avro-"
}

output "example-destination-kafka-json" {
  value = streamkap_destination_kafka.example-destination-kafka-json.id
}

output "example-destination-kafka-avro" {
  value = streamkap_destination_kafka.example-destination-kafka-avro.id
}
