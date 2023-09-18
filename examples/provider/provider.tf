terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
}

provider "streamkap" {
  host      = "https://api.streamkap.com"
  client_id = "client_id"
  secret    = "secret"
}
