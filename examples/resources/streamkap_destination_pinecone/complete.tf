# Complete Pinecone destination configuration with all options

resource "streamkap_destination_pinecone" "example" {
  name = "my-pinecone-dest"

  # Pinecone connection (required)
  pinecone_api_key    = var.pinecone_api_key
  pinecone_index_name = var.pinecone_index_name

  # Optional proxy settings
  # pinecone_proxy_host = "proxy.example.com"
  # pinecone_proxy_port = 8080

  # Collection mapping - use ${topic} for dynamic substitution
  collection_mapping = "$${topic}"

  # Schema evolution: "basic" or "none"
  schema_evolution = "basic"

  # Document ID strategy: "None", "Kafka ID", or "Field ID"
  document_id_strategy   = "Kafka ID"
  document_id_field_name = "id"

  # Vector field configuration
  vector_field_name = "vector"

  # Delete handling (requires document_id_strategy != "None")
  delete_enabled = true

  # Batch and retry settings
  batch_size       = "100"
  retry_max        = "3"
  retry_backoff_ms = "1000"
}

variable "pinecone_api_key" {
  description = "Pinecone API key"
  type        = string
  sensitive   = true
}

variable "pinecone_index_name" {
  description = "Name of the Pinecone index"
  type        = string
}
