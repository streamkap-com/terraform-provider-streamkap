# Use values from existing Streamkap connectors.
variable "source_connector" {
  type = object({
    id        = string
    name      = string
    connector = string
    topics    = set(string)
  })
  description = "Existing PostgreSQL source and selected topics, such as public.orders."
}

variable "destination" {
  type = object({
    id        = string
    name      = string
    connector = string
  })
  description = "Existing Snowflake destination."
}

resource "streamkap_pipeline" "example" {
  name = "orders-to-snowflake"

  source = {
    id        = var.source_connector.id
    name      = var.source_connector.name
    connector = var.source_connector.connector
    topics    = var.source_connector.topics
  }

  destination = {
    id        = var.destination.id
    name      = var.destination.name
    connector = var.destination.connector
  }
}
