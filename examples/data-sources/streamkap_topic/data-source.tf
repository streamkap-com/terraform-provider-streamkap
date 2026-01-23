# Look up a specific topic by ID
data "streamkap_topic" "example" {
  id = "my-source-id.public.users"
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
