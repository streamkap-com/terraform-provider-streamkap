# Complete Redis source configuration
# This example shows all available configuration options for capturing data
# from Redis streams or key patterns

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

variable "source_redis_host" {
  type        = string
  description = "The hostname of the Redis server"
}

variable "source_redis_password" {
  type        = string
  sensitive   = true
  description = "The password for Redis authentication"
}

resource "streamkap_source_redis" "example-source-redis" {
  # Display name for this source in Streamkap UI
  name = "example-source-redis"

  # Connector type: Stream or Keys
  connector_class_type = "Stream" # Options: Stream, Keys

  # Connection settings
  redis_host     = var.source_redis_host
  redis_port     = "6379"
  redis_username = "default"          # For Redis 6+ ACL
  redis_password = var.source_redis_password

  # TLS/SSL
  ssl_enabled = true

  # Stream settings
  redis_stream_name     = "orders-stream"
  redis_stream_offset   = "Latest"       # Options: Latest, Earliest
  redis_stream_delivery = "At Least Once" # Options: At Least Once, At Most Once
  redis_stream_block_seconds  = 1        # Block duration (1-60)
  redis_stream_consumer_group = "kafka-consumer-group"
  redis_stream_consumer_name  = "consumer"

  # Keys settings (when connector_class_type = "Keys")
  redis_keys_pattern         = "*"     # Key pattern to monitor
  redis_keys_timeout_seconds = 300     # Idle timeout

  # Capture mode
  mode = "LIVE" # Options: LIVE, LIVEONLY

  # Topic settings
  topic_use_stream_name = false         # Use stream name as topic
  topic                 = "redis-events" # Kafka topic name

  # Task parallelism
  tasks_max = 1 # Stream: 1-10, Keys: always 1
}

output "example-source-redis" {
  value = streamkap_source_redis.example-source-redis.id
}
