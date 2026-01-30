# Complete Enrich Async transform configuration with all options
# Enriches records asynchronously by calling external APIs using JavaScript or Python

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

resource "streamkap_transform_enrich_async" "example" {
  name = "async-enrich-orders-api"

  # Implementation language: JavaScript or Python
  transforms_language = "JavaScript"

  # Timeout for async operations in milliseconds
  transforms_async_timeout_ms = 2000

  # Parallel request capacity (1-50)
  # Formula: target_throughput_per_second * avg_api_call_time_ms / 1000
  # Example: 1000 records/sec * 10ms = 10 parallel requests
  transforms_async_capacity = 20

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for enriched records
  transforms_output_topic_pattern = "async-enriched-$0"

  # Input serialization format: Any, Avro, or Json
  transforms_input_serialization_format = "Avro"

  # Output serialization format: Any, Avro, or Json
  transforms_output_serialization_format = "Avro"
}

output "transform_enrich_async_id" {
  value = streamkap_transform_enrich_async.example.id
}
