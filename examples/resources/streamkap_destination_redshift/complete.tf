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

variable "destination_redshift_password" {
  type        = string
  sensitive   = true
  description = "Password to access the Redshift cluster"
}

# Complete Redshift destination configuration with all options
resource "streamkap_destination_redshift" "example" {
  name = "example-destination-redshift"

  # Connection settings (required)
  aws_redshift_domain   = "my-cluster.abc123xyz.us-west-2.redshift.amazonaws.com"
  aws_redshift_port     = 5439                  # Default: 5439
  aws_redshift_database = "mydb"
  connection_username   = "streamkap_user"
  connection_password   = var.destination_redshift_password

  # Schema settings (required)
  table_name_prefix = "public"                  # Schema for table names

  # Data settings
  primary_key_fields = "id"                     # Comma-separated primary key fields. Default: id
  schema_evolution   = "basic"                  # Valid values: basic, none. Default: basic

  # Performance settings
  tasks_max = 5                                 # Max active tasks (1-10). Default: 5
}

output "example_destination_redshift" {
  value = streamkap_destination_redshift.example.id
}
