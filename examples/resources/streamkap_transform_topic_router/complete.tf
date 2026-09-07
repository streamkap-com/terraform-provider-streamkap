# Complete Topic Router transform configuration with all options
# Merges several source topics into one output topic via a RegexRouter.

terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      # Set version to the release matching this documentation.
      # Beta releases require an exact prerelease version; omission selects stable.
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

resource "streamkap_transform_topic_router" "example" {
  name = "merge-order-topics"

  # Comma-separated list of input topic names to route
  transforms_input_topic_pattern = "my-source.public.orders_eu,my-source.public.orders_us"

  # Output topic replacement pattern for RegexRouter (supports $1, $2, ... capture groups)
  transforms_output_topic_pattern = "merged.orders"

  # Parallel tasks. Valid range: 1-100. Defaults to 5.
  transforms_input_job_parallelism = 2

  # Input serialization format: Any, Avro, or Json. Defaults to Any.
  transforms_input_serialization_format = "Avro"

  # Output serialization format: Any, Avro, or Json. Defaults to Any.
  transforms_output_serialization_format = "Avro"

  # Optional: auto-deploy the transform after create/update
  deploy = true

  # Optional: replay window applied on deploy
  # Valid values: "7d", "3d", "24h", "10m", "0" (continue from last position)
  replay_window = "0"
}

output "transform_topic_router_id" {
  value = streamkap_transform_topic_router.example.id
}
