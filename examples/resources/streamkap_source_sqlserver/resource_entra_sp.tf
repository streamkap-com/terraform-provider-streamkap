terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "source_sqlserver_entra_hostname" {
  type        = string
  description = "The hostname of the Azure SQL database (e.g. myserver.database.windows.net)"
}
variable "source_sqlserver_entra_tenant_id" {
  type        = string
  description = "Azure Entra tenant ID"
}
variable "source_sqlserver_entra_client_id" {
  type        = string
  description = "Azure Entra application (client) ID of the service principal"
}
variable "source_sqlserver_entra_client_secret" {
  type        = string
  sensitive   = true
  description = "Azure Entra client secret of the service principal"
}

resource "streamkap_source_sqlserver" "example-source-sqlserver-entra" {
  name                                      = "example-source-sqlserver-entra"
  database_hostname                         = var.source_sqlserver_entra_hostname
  database_port                             = 1433
  auth_mode                                 = "entra_service_principal"
  azure_tenant_id                           = var.source_sqlserver_entra_tenant_id
  azure_client_id                           = var.source_sqlserver_entra_client_id
  azure_client_secret                       = var.source_sqlserver_entra_client_secret
  database_dbname                           = "sqlserverdemo"
  schema_include_list                       = "dbo"
  table_include_list                        = "dbo.Orders"
  signal_data_collection_schema_or_database = "dbo.streamkap_signal"
  heartbeat_enabled                         = false
  binary_handling_mode                      = "bytes"
}

output "example-source-sqlserver-entra" {
  value = streamkap_source_sqlserver.example-source-sqlserver-entra.id
}
