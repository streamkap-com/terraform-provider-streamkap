terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
    }
  }
  required_version = ">= 2.0.0"
}

provider "streamkap" {}

data "streamkap_transform" "example-transform" {
  id = "660ab64aeb8783e6b76abee3"
}

output "example-transform" {
  value = data.streamkap_transform.example-transform
}