variable "host" {
  type        = string
  description = "The host of the StreamKap."
  default     = "https://aead568tp7.execute-api.us-west-2.amazonaws.com/dev"
}

variable "client_id" {
  type        = string
  description = "The client id of the service principal to use for the StreamKap."
  default     = "client_id"
  sensitive   = true
}

variable "secret_key" {
  type        = string
  description = "The client secret of the service principal to use for the StreamKap."
  default     = "secret_key"
  sensitive   = true
}

// 84ecdc52-cb29-47a7-a337-acd0ac561958
// b293f1c9-83f2-48b4-968d-b55ce6c590c4

// "fa1fcb24-d229-41f4-8ace-dbe5387a0c88"
// 3552e95b-3510-40cc-9fce-2f47602733bb