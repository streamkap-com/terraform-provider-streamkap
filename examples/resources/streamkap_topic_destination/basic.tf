variable "salesforce_source_id" {
  type = string
}

variable "destination_id" {
  type = string
}

# An API source's topic ID is source_<source id>.<connector>.<resource>.
resource "streamkap_topic_destination" "accounts" {
  topic_id       = "source_${var.salesforce_source_id}.salesforce.Account"
  destination_id = var.destination_id
}
