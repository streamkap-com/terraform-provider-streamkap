terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "~> 2.2"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "transform_id" {
  type        = string
  description = "ID of an existing Streamkap transform, from the UI or API."
}

data "streamkap_transform" "example-transform" {
  id = var.transform_id
}

output "example-transform" {
  value = data.streamkap_transform.example-transform
}