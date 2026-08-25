# Complete Kafka user configuration.
#
# Consumers need three things: READ on the topics, READ on their consumer group,
# and (when decoding Avro/Protobuf) a schema registry proxy.

resource "streamkap_kafka_user" "example" {
  # 3-24 characters, letters/digits/hyphen, no leading or trailing hyphen.
  username = "analytics-consumer"

  # 12-128 characters. Write-only: the API never returns it, so Terraform keeps
  # the configured value. Changing it rotates the credential in place.
  password = var.kafka_user_password

  # Comma-separated IPv4 addresses or CIDR ranges. Set to "" for no restriction
  # beyond the platform default.
  whitelist_ips = "10.0.0.0/8,192.168.1.5"

  # Provisions a schema registry proxy alongside the Kafka proxy and populates
  # schema_proxy_endpoint.
  is_create_schema_registry = true

  # Read every topic under the `public.` prefix.
  kafka_acls {
    topic_name            = "public."
    operation             = "READ"
    resource_pattern_type = "PREFIXED"
    resource              = "TOPIC"
  }

  # A consumer also needs to commit offsets for its group.
  kafka_acls {
    topic_name            = "analytics-consumer-group"
    operation             = "READ"
    resource_pattern_type = "LITERAL"
    resource              = "GROUP"
  }

  # Write access to one specific topic.
  # TOPIC accepts ALL, ALTER, ALTER_CONFIGS, CREATE, DELETE, DESCRIBE,
  # DESCRIBE_CONFIGS, READ, WRITE. GROUP accepts only DELETE, DESCRIBE, READ.
  kafka_acls {
    topic_name            = "analytics.results"
    operation             = "WRITE"
    resource_pattern_type = "LITERAL"
    resource              = "TOPIC"
  }
}

variable "kafka_user_password" {
  type      = string
  sensitive = true
}

output "kafka_bootstrap" {
  value = streamkap_kafka_user.example.kafka_proxy_endpoint
}

output "schema_registry_url" {
  value = streamkap_kafka_user.example.schema_proxy_endpoint
}
