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

variable "destination_redis_password" {
  type        = string
  sensitive   = true
  description = "Password to access the Redis database"
}

# Complete Redis destination configuration with all options
resource "streamkap_destination_redis" "example" {
  name = "example-destination-redis"

  # Connection settings (required)
  redis_host     = "redis.example.com"
  redis_port     = 6379                         # Default: 6379
  redis_username = "streamkap_user"
  redis_password = var.destination_redis_password

  # Security
  ssl_enabled = true                            # Enable TLS. Default: true

  # Data settings (required)
  redis_key           = "streamkap:events"      # Redis key to stream data
  redis_key_data_type = "Stream"                # Valid values: Stream, List, Hash. Default: Stream

  # Performance settings
  tasks_max = 5                                 # Max active tasks (1-10). Default: 5
}

output "example_destination_redis" {
  value = streamkap_destination_redis.example.id
}
