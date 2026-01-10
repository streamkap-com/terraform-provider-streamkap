# Minimal Rollup transform configuration
# Aggregates streaming records using SQL

resource "streamkap_transform_rollup" "example" {
  name = "rollup-orders"

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for aggregated records
  transforms_output_topic_pattern = "rollup-$0"
}
