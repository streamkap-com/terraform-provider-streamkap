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

variable "destination_kafkadirect_password" {
  type        = string
  sensitive   = true
  description = "Password for Kafka proxy authentication"
}

# Complete Kafka Direct destination configuration with all options
resource "streamkap_destination_kafkadirect" "example" {
  name = "example-destination-kafkadirect"

  # Authentication (required)
  password = var.destination_kafkadirect_password

  # Access control (required)
  whitelist_ips = "10.0.0.0/8,192.168.1.0/24,172.16.0.0/12" # Comma-separated IPs/CIDRs
}

output "example_destination_kafkadirect" {
  value = streamkap_destination_kafkadirect.example.id
}
