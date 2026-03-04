resource "streamkap_kafka_user" "example" {
  username = "my-kafka-user"
  password = var.kafka_user_password

  kafka_acls {
    topic_name            = "my-topic"
    operation             = "READ"
    resource_pattern_type = "LITERAL"
  }
}

variable "kafka_user_password" {
  type      = string
  sensitive = true
}
