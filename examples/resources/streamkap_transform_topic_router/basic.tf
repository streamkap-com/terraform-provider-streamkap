# Minimal Topic Router transform configuration
# Rewrites topic names with a RegexRouter — no user code, so this transform has
# no transforms_language / implementation_json to supply.

resource "streamkap_transform_topic_router" "example" {
  name = "route-orders"

  # Comma-separated list of input topic names to route
  transforms_input_topic_pattern = "my-source.public.orders"

  # Output topic replacement pattern for RegexRouter
  transforms_output_topic_pattern = "merged.orders"
}
