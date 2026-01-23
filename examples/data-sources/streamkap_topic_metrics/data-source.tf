# Get metrics for specific topics
# Note: Use streamkap_topics data source first to get topic_db_ids
data "streamkap_topic_metrics" "example" {
  entities = [
    {
      id           = "source-123"
      entity_type  = "sources"
      connector    = "postgresql"
      topic_ids    = ["source-123.public.users", "source-123.public.orders"]
      topic_db_ids = ["64abc123def456789012345a", "64abc123def456789012345b"]
    },
    {
      id           = "transform-456"
      entity_type  = "transforms"
      connector    = "map_filter"
      topic_ids    = ["transform-456.filtered_users"]
      topic_db_ids = ["64abc123def456789012345c"]
    }
  ]

  time_interval = 24
  time_unit     = "hours"
}

# Output metrics
output "topic_throughput" {
  value = {
    for r in data.streamkap_topic_metrics.example.results :
    r.topic_id => {
      messages_in  = r.messages_in
      messages_out = r.messages_out
      lag          = r.lag
    }
  }
}
