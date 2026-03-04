data "streamkap_roles" "all" {}

resource "streamkap_client_credential" "example" {
  role_ids = [data.streamkap_roles.all.roles[0].id]
}

output "client_id" {
  value = streamkap_client_credential.example.client_id
}

output "secret" {
  value     = streamkap_client_credential.example.secret
  sensitive = true
}
