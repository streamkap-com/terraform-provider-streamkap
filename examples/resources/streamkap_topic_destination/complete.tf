variable "api_source_topic_ids" {
  type = set(string)
}

variable "destination_id" {
  type = string
}

resource "streamkap_topic_destination" "example" {
  for_each       = var.api_source_topic_ids
  topic_id       = each.value
  destination_id = var.destination_id
}
