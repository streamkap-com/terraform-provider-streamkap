# Complete topic configuration with all options

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

# Topic partition management
# Use this to adjust the number of partitions for a Streamkap topic
resource "streamkap_topic" "example" {
  # Topic ID format: source_{source_id}.{schema}.{table}
  topic_id = "source_abc123def456.public.users"

  # Number of partitions (must be >= current count, cannot decrease)
  partition_count = 12
}

output "topic_id" {
  description = "The topic identifier"
  value       = streamkap_topic.example.topic_id
}

output "partition_count" {
  description = "Current partition count"
  value       = streamkap_topic.example.partition_count
}
