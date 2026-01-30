# Complete Rollup transform configuration with all options
# Aggregates streaming records into summaries using SQL

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

resource "streamkap_transform_rollup" "example" {
  name = "rollup-daily-sales"

  # Language for the transform (SQL only)
  transforms_language = "SQL"

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for aggregated records
  transforms_output_topic_pattern = "rollup-$0"

  # Input serialization format: Any, Avro, or Json
  transforms_input_serialization_format = "Avro"

  # Output serialization format: Any, Avro, or Json
  transforms_output_serialization_format = "Avro"
}

output "transform_rollup_id" {
  value = streamkap_transform_rollup.example.id
}
