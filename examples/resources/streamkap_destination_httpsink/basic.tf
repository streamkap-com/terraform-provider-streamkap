# Minimal HTTP Sink destination configuration

resource "streamkap_destination_httpsink" "example" {
  name     = "my-httpsink-dest"
  http_url = var.http_url
}

variable "http_url" {
  description = "HTTP endpoint URL"
  type        = string
}
