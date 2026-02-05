# Enrich transform with implementation_json
# Enrich streaming records by joining with reference data

terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

resource "streamkap_transform_enrich" "example" {
  name = "enrich-orders-with-locations"

  transforms_language             = "SQL"
  transforms_input_topic_pattern  = "my-source\\.public\\.orders"
  transforms_output_topic_pattern = "enriched-$0"

  # Manage enrichment implementation via Terraform
  # Define main table (streaming), lookup table (reference), and enrich SQL
  implementation_json = jsonencode({
    mainTable = {
      name              = "orders"
      topicMatcherRegex = ".*orders$"
      createTableSQL    = <<-SQL
        CREATE TABLE orders (
          order_id STRING PRIMARY KEY,
          location_id STRING,
          total_amount DECIMAL(10,2)
        )
      SQL
    }
    lookupTable = {
      name              = "locations"
      topicMatcherRegex = ".*locations$"
      createTableSQL    = <<-SQL
        CREATE TABLE locations (
          location_id STRING PRIMARY KEY,
          city STRING,
          country STRING
        )
      SQL
    }
    enrichSQL = <<-SQL
      SELECT
        o.order_id,
        o.total_amount,
        l.city,
        l.country
      FROM orders o
      JOIN locations l ON o.location_id = l.location_id
    SQL
  })
}

output "transform_id" {
  value = streamkap_transform_enrich.example.id
}
