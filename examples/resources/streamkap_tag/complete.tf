# Complete tag configuration

resource "streamkap_tag" "example" {
  name        = "production"
  description = "Tag for production environment resources"

  # Entity types this tag can be applied to
  # Valid values: sources, destinations, pipelines
  type = ["sources", "destinations", "pipelines"]
}

output "tag_id" {
  value = streamkap_tag.example.id
}
