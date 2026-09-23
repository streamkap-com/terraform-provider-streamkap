terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "3.0.1"
    }
  }
}

provider "streamkap" {}
