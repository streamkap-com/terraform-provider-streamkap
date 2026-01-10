# Minimal Azure Blob Storage destination configuration

resource "streamkap_destination_azblob" "example" {
  name                     = "my-azblob-dest"
  azblob_connection_string = var.azblob_connection_string
  azblob_container_name    = var.azblob_container_name
}

variable "azblob_connection_string" {
  description = "Azure Blob Storage connection string"
  type        = string
  sensitive   = true
}

variable "azblob_container_name" {
  description = "Azure Blob Storage container name"
  type        = string
}
