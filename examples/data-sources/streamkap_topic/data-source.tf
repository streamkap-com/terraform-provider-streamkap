variable "topic_id" {
  type        = string
  description = "ID of an existing Streamkap topic."
}

# Look up a specific topic by ID
data "streamkap_topic" "example" {
  id = var.topic_id
}

# Use topic information
output "topic_partitions" {
  value = data.streamkap_topic.example.partitions
}

output "topic_entity" {
  value = {
    id   = data.streamkap_topic.example.entity_id
    name = data.streamkap_topic.example.entity_name
    type = data.streamkap_topic.example.entity_type
  }
}

output "topic_kafka_config" {
  value = {
    retention_ms   = data.streamkap_topic.example.retention_ms
    cleanup_policy = data.streamkap_topic.example.cleanup_policy
  }
}
