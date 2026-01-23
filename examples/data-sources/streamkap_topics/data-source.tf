# List all topics
data "streamkap_topics" "all" {}

# List topics from sources only
data "streamkap_topics" "source_topics" {
  entity_type = "sources"
}

# List topics from specific entities
data "streamkap_topics" "filtered" {
  entity_type = "sources"
  entity_ids  = ["source-id-1", "source-id-2"]
}

# Output topic count
output "total_topics" {
  value = data.streamkap_topics.all.total
}

# Output topic names
output "topic_names" {
  value = [for t in data.streamkap_topics.all.topics : t.name]
}
