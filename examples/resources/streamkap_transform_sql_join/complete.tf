# Complete SQL Join transform configuration with all options
# Joins multiple streaming topics using SQL with configurable state TTL

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

resource "streamkap_transform_sql_join" "example" {
  name = "join-orders-with-customers"

  # Language for the transform (SQL only)
  transforms_language = "SQL"

  # Regex pattern to match multiple input topics for joining
  transforms_input_topic_pattern = "my-source\\.public\\.(orders|customers|products)"

  # Output topic pattern for joined records
  transforms_output_topic_pattern = "joined-$0"

  # State TTL - how long to retain state for joins
  # Valid duration formats: 3d, 10m, 2h, 30s
  transforms_topic_ttl = "7d"

  # Input serialization format: Any, Avro, or Json
  transforms_input_serialization_format = "Avro"

  # Output serialization format: Any, Avro, or Json
  transforms_output_serialization_format = "Avro"
}

output "transform_sql_join_id" {
  value = streamkap_transform_sql_join.example.id
}
