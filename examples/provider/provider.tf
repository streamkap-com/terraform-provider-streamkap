terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "3.1.0"
    }
  }
}

provider "streamkap" {}
