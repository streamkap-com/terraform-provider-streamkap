# Minimal client credential (API token) for machine-to-machine authentication.
# Look role IDs up rather than hardcoding them.

data "streamkap_roles" "all" {}

resource "streamkap_client_credential" "example" {
  role_ids    = [data.streamkap_roles.all.roles[0].id]
  description = "CI pipeline token"
}

# The secret is returned once, at creation, and stored in state. Later reads
# echo a masked value, so this is the only chance to capture it.
output "client_id" {
  value = streamkap_client_credential.example.client_id
}

output "client_secret" {
  value     = streamkap_client_credential.example.secret
  sensitive = true
}
