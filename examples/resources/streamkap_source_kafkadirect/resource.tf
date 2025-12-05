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

resource "streamkap_source_kafkadirect" "example-source-kafkadirect" {
  name                            = "example-source-kafkadirect"
  topic_prefix                    = "myapp"
  topic_include_list_user_defined = "orders,customers,products"
  format                          = "json"
  schemas_enable                  = false
}

output "example-source-kafkadirect" {
  value = streamkap_source_kafkadirect.example-source-kafkadirect.id
}
