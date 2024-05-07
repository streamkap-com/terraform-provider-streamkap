variable "host" {
  type        = string
  description = "The host of the StreamKap."
  default     = "https://api.streamkap.com"
}

variable "client_id" {
  type        = string
  description = "The client id of the service principal to use for the StreamKap."
  default     = "5a0ed5a9-1bb5-4339-9e9a-186404302cae"
  sensitive   = true
}

variable "secret_key" {
  type        = string
  description = "The client secret of the service principal to use for the StreamKap."
  default     = "6b362b0b-f5df-4c05-9cea-ba85cfc5e1f9"
  sensitive   = true
}

variable "source_host" {
  type        = string
  description = "The host of the source to use for the StreamKap."
  default     = "https://source.streamkap.com"
}

variable "source_password" {
  type        = string
  description = "The password of the source to use for the StreamKap."
  default     = "source_password"
  sensitive   = true
}

variable "destination_host" {
  type        = string
  description = "The host of the destination to use for the StreamKap."
  default     = "https://destination.streamkap.com"
}

variable "destination_private_key" {
  type        = string
  description = "The private key to authenticate the user."
  default     = "my private key"
  sensitive   = true
}

variable "destination_key_passphrase" {
  type        = string
  description = "The private key passphrase to authenticate the user."
  default     = "my private key passphrase"
  sensitive   = true
}
