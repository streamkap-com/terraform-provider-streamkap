# Minimal Elasticsearch source configuration
# This example captures data from Elasticsearch indices

resource "streamkap_source_elasticsearch" "example" {
  name = "my-elasticsearch-source"

  # Connection details
  es_host          = "elasticsearch.example.com"
  es_port          = "443"
  http_auth_user   = "elastic"
  http_auth_password = var.es_password

  # Indices to capture
  endpoint_include_list = "orders,customers"

  # Datetime field for incremental sync
  datetime_field_name = "updated_at"
}

variable "es_password" {
  description = "Elasticsearch password"
  type        = string
  sensitive   = true
}
