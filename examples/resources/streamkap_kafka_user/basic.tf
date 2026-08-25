# Minimal Kafka user: read access to a single topic.
# The username is the resource ID and cannot be changed in place.

resource "streamkap_kafka_user" "example" {
  username = "analytics-reader"
  password = var.kafka_user_password

  kafka_acls {
    topic_name            = "public.customers"
    operation             = "READ"
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
