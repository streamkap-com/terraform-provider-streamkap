# Minimal Enrich transform configuration
# Enriches records by joining with reference data using SQL

resource "streamkap_transform_enrich" "example" {
  name = "enrich-orders"

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for enriched records
  transforms_output_topic_pattern = "enriched-$0"
}
