terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
    }
  }
}

provider "streamkap" {}

resource "streamkap_kafka_user" "example" {
  username = "my-kafka-user"
  password = var.kafka_user_password

  # Optional: comma-separated IP whitelist
  whitelist_ips = "192.168.1.0/24,10.0.0.1"

  # Optional: create a schema registry user alongside
  is_create_schema_registry = true

  # ACL rules for topic access
  kafka_acls {
    topic_name            = "my-topic"
    operation             = "READ"
    resource_pattern_type = "LITERAL"
    resource              = "TOPIC"
  }

  kafka_acls {
    topic_name            = "my-topic-prefix"
    operation             = "ALL"
    resource_pattern_type = "PREFIXED"
    resource              = "TOPIC"
  }

  kafka_acls {
    topic_name            = "my-consumer-group"
    operation             = "READ"
    resource_pattern_type = "LITERAL"
    resource              = "GROUP"
  }
}

variable "kafka_user_password" {
  type      = string
  sensitive = true
}

output "kafka_proxy_endpoint" {
  value = streamkap_kafka_user.example.kafka_proxy_endpoint
}

output "schema_proxy_endpoint" {
  value = streamkap_kafka_user.example.schema_proxy_endpoint
}
