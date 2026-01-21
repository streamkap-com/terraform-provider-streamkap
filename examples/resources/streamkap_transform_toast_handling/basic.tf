# Minimal TOAST Handling transform configuration
# Handles PostgreSQL TOAST (The Oversized-Attribute Storage Technique) columns
# by reconstructing large values that were stored externally

resource "streamkap_transform_toast_handling" "example" {
  name = "toast-handling-orders"

  # Regex pattern to match input topics containing TOAST data
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for processed records with TOAST values reconstructed
  transforms_output_topic_pattern = "toast-handled-$0"
}
