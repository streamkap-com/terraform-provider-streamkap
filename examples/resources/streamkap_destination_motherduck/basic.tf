# Minimal Motherduck destination configuration

resource "streamkap_destination_motherduck" "example" {
  name              = "my-motherduck-dest"
  motherduck_token  = var.motherduck_token
  motherduck_catalog = var.motherduck_catalog
}

variable "motherduck_token" {
  description = "Motherduck API token"
  type        = string
  sensitive   = true
}

variable "motherduck_catalog" {
  description = "Motherduck database/catalog name"
  type        = string
}
