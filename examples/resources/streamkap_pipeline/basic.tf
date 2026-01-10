# Minimal pipeline configuration
# Connects a source to a destination without transforms

resource "streamkap_pipeline" "example" {
  name = "my-pipeline"

  # Source connector (must be created first)
  source = {
    id        = streamkap_source_postgresql.example.id
    name      = streamkap_source_postgresql.example.name
    connector = streamkap_source_postgresql.example.connector
    topics    = ["public.orders", "public.customers"]
  }

  # Destination connector (must be created first)
  destination = {
    id        = streamkap_destination_snowflake.example.id
    name      = streamkap_destination_snowflake.example.name
    connector = streamkap_destination_snowflake.example.connector
  }
}
