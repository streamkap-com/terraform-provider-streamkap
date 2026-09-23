# List all topics
data "streamkap_topics" "all" {}

# List topics from sources only
data "streamkap_topics" "source_topics" {
  entity_type = "sources"
}

variable "source_ids" {
  type        = list(string)
  description = "Streamkap source IDs to filter by."
}

# List topics from specific entities
data "streamkap_topics" "filtered" {
  entity_type = "sources"
  entity_ids  = var.source_ids
}

# Output topic count
output "total_topics" {
  value = data.streamkap_topics.all.total
}

# Output topic names
output "topic_names" {
  value = [for t in data.streamkap_topics.all.topics : t.name]
}
