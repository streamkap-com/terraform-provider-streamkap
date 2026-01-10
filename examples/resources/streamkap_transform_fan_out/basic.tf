# Minimal Fan Out transform configuration
# Splits a single record into multiple output records using JavaScript

resource "streamkap_transform_fan_out" "example" {
  name = "fanout-events"

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.events"

  # Output topic pattern for fanned-out records
  transforms_output_topic_pattern = "fanout-$0"
}
