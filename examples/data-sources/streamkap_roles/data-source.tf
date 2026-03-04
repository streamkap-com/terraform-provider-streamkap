data "streamkap_roles" "all" {}

output "available_roles" {
  value = data.streamkap_roles.all.roles
}
