# Minimal Enrich Async transform configuration
# Enriches records asynchronously using external API calls (JavaScript or Python)

resource "streamkap_transform_enrich_async" "example" {
  name = "async-enrich-orders"

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for enriched records
  transforms_output_topic_pattern = "async-enriched-$0"
}
