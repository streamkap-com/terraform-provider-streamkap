# Minimal Redis destination configuration

resource "streamkap_destination_redis" "example" {
  name           = "my-redis-dest"
  redis_host     = var.redis_host
  redis_password = var.redis_password
}

variable "redis_host" {
  description = "Redis server hostname"
  type        = string
}

variable "redis_password" {
  description = "Password to access Redis"
  type        = string
  sensitive   = true
}
