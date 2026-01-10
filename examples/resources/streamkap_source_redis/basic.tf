# Minimal Redis source configuration
# This example captures data from Redis streams

resource "streamkap_source_redis" "example" {
  name = "my-redis-source"

  # Connection details
  redis_host     = "redis.example.com"
  redis_port     = "6379"
  redis_password = var.redis_password

  # Stream to capture
  redis_stream_name = "orders-stream"

  # Kafka topic for output
  topic = "redis-orders"
}

variable "redis_password" {
  description = "Redis password"
  type        = string
  sensitive   = true
}
