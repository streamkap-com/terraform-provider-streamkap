variable "api_source_topic_id" {
  type = string
}

variable "destination_id" {
  type = string
}

resource "streamkap_topic_destination" "example" {
  topic_id       = var.api_source_topic_id
  destination_id = var.destination_id
}
