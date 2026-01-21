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

variable "destination_weaviate_api_key" {
  type        = string
  sensitive   = true
  description = "Weaviate API key for authentication"
}

resource "streamkap_destination_weaviate" "example-destination-weaviate" {
  name = "example-destination-weaviate"

  # Connection settings
  weaviate_connection_url = "https://my-cluster.weaviate.network"
  weaviate_grpc_url       = "my-cluster.weaviate.network:50051"
  weaviate_grpc_secured   = true

  # Authentication (API_KEY, OIDC_CLIENT_CREDENTIALS, or NONE)
  weaviate_auth_scheme = "API_KEY"
  weaviate_api_key     = var.destination_weaviate_api_key

  # Collection mapping - use ${topic} for dynamic topic name substitution
  collection_mapping = "$${topic}"

  # Schema evolution (basic or none)
  schema_evolution = "basic"

  # Document ID configuration
  document_id_strategy   = "Field ID"
  document_id_field_name = "id"

  # Vector configuration
  vector_strategy   = "None"
  vector_field_name = "vector"

  # Delete handling
  delete_enabled = true

  # Batch settings
  batch_size           = "100"
  pool_size            = "2"
  await_termination_ms = "10000"

  # Retry settings
  max_connection_retries = "3"
  max_timeout_retries    = "3"
  retry_interval         = "2000"
  retry_backoff_ms       = "1000"

  # Consistency level (ALL, ONE, QUORUM)
  consistency_level = "QUORUM"
}

output "example-destination-weaviate" {
  value = streamkap_destination_weaviate.example-destination-weaviate.id
}
