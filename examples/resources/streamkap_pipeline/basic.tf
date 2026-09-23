# Use IDs and names from existing Streamkap connectors.
variable "source_connector" {
  type = object({
    id     = string
    name   = string
    topics = set(string)
  })
  description = "Existing PostgreSQL source and selected topics, such as public.orders."
}

variable "destination" {
  type = object({
    id   = string
    name = string
  })
  description = "Existing Snowflake destination."
}

resource "streamkap_pipeline" "example" {
  name = "orders-to-snowflake"

  source = {
    id        = var.source_connector.id
    name      = var.source_connector.name
    connector = "postgresql"
    topics    = var.source_connector.topics
  }

  destination = {
    id        = var.destination.id
    name      = var.destination.name
    connector = "snowflake"
  }
}
