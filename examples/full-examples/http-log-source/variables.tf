variable "token" {
  description = "Panther API token"
  type        = string
}

variable "url" {
  description = "Panther API URL"
  type        = string
}

variable "integration_label" {
  description = "The name of the integration."
  type        = string
}

variable "log_stream_type" {
  description = "Type of log stream."
  type        = string
}

variable "log_types" {
  description = "List of log types for the HTTP source."
  type = list(string)
}

variable "security_type" {
  description = "Type of security used."
  type        = string
}

variable "security_header_key" {
  description = "Key for the security header."
  type        = string
}

variable "security_secret_value" {
  description = "Secret value."
  type        = string
  sensitive   = true
}

variable "security_username" {
  description = "Username for security purposes."
  type        = string
}

variable "security_password" {
  description = "Password for security purposes."
  type        = string
  sensitive   = true
}

variable "security_alg" {
  description = "Security algorithm used."
  type        = string
}
