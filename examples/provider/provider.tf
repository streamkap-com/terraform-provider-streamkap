terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.1.19"
    }
  }
}

provider "streamkap" {}
