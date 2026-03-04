terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
    }
  }
}

provider "streamkap" {}

# Look up available roles
data "streamkap_roles" "all" {}

resource "streamkap_client_credential" "example" {
  # At least one role ID is required
  role_ids = [data.streamkap_roles.all.roles[0].id]

  # Optional: description for this credential
  description = "API token for CI/CD pipeline"

  # Optional: service ID to associate
  service_id = "my-service"
}

output "client_id" {
  value = streamkap_client_credential.example.client_id
}

output "secret" {
  value     = streamkap_client_credential.example.secret
  sensitive = true
}

output "created_at" {
  value = streamkap_client_credential.example.created_at
}
