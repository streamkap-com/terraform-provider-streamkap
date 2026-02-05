# Rollup transform with implementation_json
# Aggregate streaming records over time windows using SQL

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

resource "streamkap_transform_rollup" "example" {
  name = "rollup-daily-sales"

  transforms_language             = "SQL"
  transforms_input_topic_pattern  = "my-source\\.public\\.orders"
  transforms_output_topic_pattern = "aggregated-$0"

  # Manage rollup aggregation implementation via Terraform
  implementation_json = jsonencode({
    inputTables = [
      {
        name              = "orders"
        topicMatcherRegex = ".*orders$"
        createTableSQL    = <<-SQL
          CREATE TABLE orders (
            product_id STRING,
            quantity INT,
            amount DECIMAL(10,2),
            order_date DATE
          )
        SQL
      }
    ]
    rollupSQL = <<-SQL
      SELECT
        product_id,
        order_date,
        SUM(quantity) AS total_quantity,
        SUM(amount) AS total_amount,
        COUNT(*) AS order_count
      FROM orders
      GROUP BY product_id, order_date
    SQL
    keyFields           = ["product_id", "order_date"]
    sourceIdleTimeoutMs = 30000     # 30 seconds idle timeout
    stateTTLMs          = 86400000  # 24 hours state retention
  })
}

output "transform_id" {
  value = streamkap_transform_rollup.example.id
}
