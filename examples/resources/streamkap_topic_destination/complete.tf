variable "destination_id" {
  type = string
}

# One link per resource of the source. Referencing the source also makes
# Terraform remove the links before it deletes the source.
resource "streamkap_topic_destination" "salesforce" {
  for_each       = toset(streamkap_source_salesforce.example.resources)
  topic_id       = "source_${streamkap_source_salesforce.example.id}.salesforce.${each.value}"
  destination_id = var.destination_id
}
