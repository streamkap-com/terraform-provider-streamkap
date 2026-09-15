# Minimal Redshift destination configuration

resource "streamkap_destination_redshift" "example" {
  name                  = "my-redshift-dest"
  aws_redshift_domain   = var.redshift_domain
  aws_redshift_database = "mydb"
  connection_username   = var.redshift_username
  connection_password   = var.redshift_password
  table_name_prefix     = "public"
}

variable "redshift_domain" {
  description = "Redshift cluster domain"
  type        = string
}

variable "redshift_username" {
  description = "Username to access Redshift"
  type        = string
}

variable "redshift_password" {
  description = "Password to access Redshift"
  type        = string
  sensitive   = true
}
