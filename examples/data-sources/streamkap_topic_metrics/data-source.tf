variable "source_id" {
  type        = string
  description = "ID of the PostgreSQL source to query."
}

variable "source_topics" {
  type = list(object({
    topic_id    = string
    topic_db_id = string
  }))
  description = "Kafka topic names and matching topic database IDs from the Streamkap API."
}

data "streamkap_topic_metrics" "example" {
  entities = [{
    id           = var.source_id
    entity_type  = "sources"
    connector    = "postgresql"
    topic_ids    = [for topic in var.source_topics : topic.topic_id]
    topic_db_ids = [for topic in var.source_topics : topic.topic_db_id]
  }]
}

# Output metrics
output "topic_status" {
  value = {
    for r in data.streamkap_topic_metrics.example.results :
    r.topic_id => {
      partition_count        = r.partition_count
      replication_factor     = r.replication_factor
      retention_ms           = r.retention_ms
      last_message_timestamp = r.last_message_timestamp
      snapshot_status        = jsondecode(r.snapshot_status_json)
      record_error_total     = r.record_error_total
    }
  }
}
