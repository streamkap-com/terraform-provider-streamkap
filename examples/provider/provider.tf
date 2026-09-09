terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "3.0.0-beta.31"
    }
  }
}

provider "streamkap" {}
