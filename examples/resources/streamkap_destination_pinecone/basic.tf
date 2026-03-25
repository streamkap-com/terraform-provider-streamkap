# Minimal Pinecone destination configuration

resource "streamkap_destination_pinecone" "example" {
  name = "my-pinecone-dest"

  # Pinecone connection
  pinecone_api_key    = var.pinecone_api_key
  pinecone_index_name = var.pinecone_index_name
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
