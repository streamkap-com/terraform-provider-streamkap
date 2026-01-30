# Minimal tag configuration
# Tags help organize and categorize resources in Streamkap

resource "streamkap_tag" "example" {
  name = "production"
  type = ["sources", "destinations", "pipelines"]
}
