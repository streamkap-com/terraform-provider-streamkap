# Minimal Map/Filter transform configuration
# Filters and transforms records from source topics using JavaScript or Python

resource "streamkap_transform_map_filter" "example" {
  name = "filter-orders"

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern (replacement string)
  transforms_output_topic_pattern = "filtered-$0"
}
