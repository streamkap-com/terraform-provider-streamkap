# Minimal Weaviate destination configuration

resource "streamkap_destination_weaviate" "example" {
  name = "my-weaviate-dest"

  # Weaviate connection
  weaviate_connection_url = var.weaviate_url
  weaviate_grpc_url       = var.weaviate_grpc_url
}

variable "weaviate_url" {
  description = "Weaviate connection URL (e.g., http://localhost:8080)"
  type        = string
  default     = "http://localhost:8080"
}

variable "weaviate_grpc_url" {
  description = "Weaviate gRPC connection URL (e.g., localhost:50051)"
  type        = string
  default     = "localhost:50051"
}
