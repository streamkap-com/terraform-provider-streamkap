# Minimal Kafka destination configuration

resource "streamkap_destination_kafka" "example" {
  name                 = "my-kafka-dest"
  kafka_sink_bootstrap = var.kafka_bootstrap_servers
}

variable "kafka_bootstrap_servers" {
  description = "Kafka bootstrap servers (e.g., host1:port1,host2:port2)"
  type        = string
}
