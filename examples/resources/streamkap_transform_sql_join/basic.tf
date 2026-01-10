# Minimal SQL Join transform configuration
# Joins multiple streaming topics using SQL

resource "streamkap_transform_sql_join" "example" {
  name = "join-orders-customers"

  # Regex pattern to match input topics (multiple topics for joins)
  transforms_input_topic_pattern = "my-source\\.public\\.(orders|customers)"

  # Output topic pattern for joined records
  transforms_output_topic_pattern = "joined-$0"
}
