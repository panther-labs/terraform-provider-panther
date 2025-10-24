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
  type        = list(string)
}

variable "auth_method" {
  description = "Authentication method used."
  type        = string
}

variable "auth_header_key" {
  description = "Key for the authentication header."
  type        = string
}

variable "auth_secret_value" {
  description = "Authentication secret value."
  type        = string
  sensitive   = true
}

variable "json_array_envelope_field" {
  description = "Envelope field for json array stream"
  type        = string
}

variable "xml_root_element" {
  description = "Root element for xml stream"
  type        = string
}
