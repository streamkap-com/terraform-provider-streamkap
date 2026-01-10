# Complete Map/Filter transform configuration with all options
# Transform/Filter Records allows filtering and transforming data in real-time

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

resource "streamkap_transform_map_filter" "example" {
  name = "transform-orders"

  # Implementation language: JavaScript or Python
  transforms_language = "JavaScript"

  # Regex pattern to match input topics
  # Use regex syntax to match topic names from your sources
  transforms_input_topic_pattern = "my-source\\.public\\.(orders|customers)"

  # Output topic pattern for transformed records
  # $0 refers to the matched input topic name
  transforms_output_topic_pattern = "transformed-$0"

  # Input serialization format: Any, Avro, or Json
  transforms_input_serialization_format = "Avro"

  # Output serialization format: Any, Avro, or Json
  transforms_output_serialization_format = "Avro"
}

output "transform_map_filter_id" {
  value = streamkap_transform_map_filter.example.id
}
