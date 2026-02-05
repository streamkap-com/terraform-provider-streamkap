# SQL Join transform with implementation_json
# Join multiple streaming topics using SQL with configurable state management

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

resource "streamkap_transform_sql_join" "example" {
  name = "join-orders-customers"

  transforms_language             = "SQL"
  transforms_input_topic_pattern  = "my-source\\.public\\.(orders|customers)"
  transforms_output_topic_pattern = "joined-$0"
  transforms_topic_ttl            = "7d"

  # Manage SQL join implementation via Terraform
  # This includes table definitions and the join SQL query
  implementation_json = jsonencode({
    inputTables = [
      {
        name              = "orders"
        topicMatcherRegex = ".*orders$"
        createTableSQL    = <<-SQL
          CREATE TABLE orders (
            order_id STRING PRIMARY KEY,
            customer_id STRING,
            total_amount DECIMAL(10,2),
            order_date TIMESTAMP
          )
        SQL
      },
      {
        name              = "customers"
        topicMatcherRegex = ".*customers$"
        createTableSQL    = <<-SQL
          CREATE TABLE customers (
            customer_id STRING PRIMARY KEY,
            name STRING,
            email STRING,
            created_at TIMESTAMP
          )
        SQL
      }
    ]
    joinSQL = <<-SQL
      SELECT
        o.order_id,
        o.total_amount,
        o.order_date,
        c.name AS customer_name,
        c.email AS customer_email
      FROM orders o
      JOIN customers c ON o.customer_id = c.customer_id
    SQL
    keyFields   = ["order_id"]
    stateTtlMs  = "604800000"  # 7 days in milliseconds
  })
}

output "transform_id" {
  value = streamkap_transform_sql_join.example.id
}
