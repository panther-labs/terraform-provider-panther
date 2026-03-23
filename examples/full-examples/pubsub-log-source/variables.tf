variable "token" {
  description = "Panther API token"
  type        = string
}

variable "url" {
  description = "Panther API URL"
  type        = string
}

variable "integration_label" {
  description = "The name of the Pub/Sub log source integration."
  type        = string
}

variable "subscription_id" {
  description = "The GCP Pub/Sub subscription ID."
  type        = string
}

variable "project_id" {
  description = "The GCP project ID. Optional for service_account credentials (derived from the keyfile). Required for WIF."
  type        = string
  default     = ""
}

variable "credentials_file" {
  description = "Path to the GCP credentials JSON file (service account key or WIF config)."
  type        = string
  sensitive   = true
}

variable "credentials_type" {
  description = "The type of GCP credentials: service_account or wif."
  type        = string
}

variable "log_types" {
  description = "List of log types for the Pub/Sub source."
  type        = list(string)
}

variable "log_stream_type" {
  description = "Type of log stream."
  type        = string
  default     = "Auto"
}

variable "json_array_envelope_field" {
  description = "Envelope field for json array stream"
  type        = string
  default     = ""
}
