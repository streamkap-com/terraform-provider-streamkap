# Minimal Kafka Direct destination configuration

resource "streamkap_destination_kafkadirect" "example" {
  name          = "my-kafkadirect-dest"
  password      = var.kafka_password
  whitelist_ips = var.whitelist_ips
}

variable "kafka_password" {
  description = "Password for the Kafka proxy"
  type        = string
  sensitive   = true
}

variable "whitelist_ips" {
  description = "Comma-separated list of IPs/CIDRs to whitelist"
  type        = string
}
