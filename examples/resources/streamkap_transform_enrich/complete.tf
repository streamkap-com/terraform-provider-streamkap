# Complete Enrich transform configuration with all options
# Enriches streaming records by joining with reference data using SQL

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

resource "streamkap_transform_enrich" "example" {
  name = "enrich-orders-with-customers"

  # Language for the transform (SQL only for Enrich)
  transforms_language = "SQL"

  # Regex pattern to match input topics from your sources
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for enriched records
  # $0 refers to the matched input topic name
  transforms_output_topic_pattern = "enriched-$0"
}

output "transform_enrich_id" {
  value = streamkap_transform_enrich.example.id
}
