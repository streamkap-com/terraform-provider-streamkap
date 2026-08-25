# Complete client credential configuration.

data "streamkap_roles" "all" {}

# Pick roles by key instead of by position, so the config does not depend on the
# order the API happens to return them in.
locals {
  role_ids_by_key = { for role in data.streamkap_roles.all.roles : role.key => role.id }
}

resource "streamkap_client_credential" "example" {
  # At least one role is required. Changing this updates the credential in
  # place — the secret is not rotated.
  role_ids = [local.role_ids_by_key["ReadOnly"]]

  # Also updatable in place. Once set it can be replaced but not cleared.
  description = "Read-only token for the metrics exporter"

  # Optional. The update endpoint does not accept service_id, so changing it
  # replaces the credential and issues a new secret.
  service_id = var.service_id
}

variable "service_id" {
  type    = string
  default = null
}

output "available_role_keys" {
  value = [for role in data.streamkap_roles.all.roles : role.key]
}

# Roles resolved by the server for this credential.
output "granted_roles" {
  value = [for role in streamkap_client_credential.example.roles : role.name]
}

output "client_id" {
  value = streamkap_client_credential.example.client_id
}

output "client_secret" {
  value     = streamkap_client_credential.example.secret
  sensitive = true
}
