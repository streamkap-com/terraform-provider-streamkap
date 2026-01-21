# Minimal Un-Nesting transform configuration
# Flattens arrays and nested records into separate output records

resource "streamkap_transform_un_nesting" "example" {
  name = "unnest-order-items"

  # Regex pattern to match input topics with nested data
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for flattened records
  transforms_output_topic_pattern = "unnested-$0"
}
