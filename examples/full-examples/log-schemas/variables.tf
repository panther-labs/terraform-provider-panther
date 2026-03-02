variable "token" {
  description = "Panther API token"
  type        = string
  sensitive   = true
}

variable "url" {
  description = "Panther API URL"
  type        = string
}