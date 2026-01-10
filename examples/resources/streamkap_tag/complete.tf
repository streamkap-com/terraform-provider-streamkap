# Complete tag configuration with all options

resource "streamkap_tag" "example" {
  name        = "production"
  description = "Tag for production environment resources"

  # Entity types this tag can be applied to
  # Valid values: sources, destinations, pipelines
  type = ["sources", "destinations", "pipelines"]
}

# Example: Using tags with resources
resource "streamkap_pipeline" "example" {
  name = "my-pipeline"

  # Apply the tag to this pipeline
  tags = [streamkap_tag.example.id]

  # ... other configuration
}
