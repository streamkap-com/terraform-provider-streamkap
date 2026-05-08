resource "streamkap_source_informix" "example" {
  name              = "my-informix-source"
  database_hostname = "informix.example.com"
  database_user     = "streamkap"
  database_password = var.informix_password
  database_dbname   = "appdb"

  schema_include_list = "public"
  table_include_list  = "public.orders,public.customers"
}

variable "informix_password" {
  type      = string
  sensitive = true
}
