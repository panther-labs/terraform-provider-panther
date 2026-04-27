variable "token" {
  description = "Panther API token"
  type        = string
  sensitive   = true
}

variable "url" {
  description = "Panther API URL"
  type        = string
}

# --- GCS source (parent) variables ---

variable "integration_label" {
  description = "Display name for the GCS log source integration"
  type        = string
  default     = "provider-log-source-alarm-test"
}

variable "project_id" {
  description = "GCP project ID. Optional for service_account credentials (derived from the keyfile); required for WIF."
  type        = string
  default     = ""
}

variable "subscription_id" {
  description = "Pub/Sub subscription for GCS bucket notifications"
  type        = string
}

variable "gcs_bucket" {
  description = "GCS bucket name"
  type        = string
}

variable "credentials_file" {
  description = "Path to the GCP service-account JSON keyfile (or WIF credential config)"
  type        = string
}

variable "credentials_type" {
  description = "One of \"service_account\" or \"wif\""
  type        = string
  default     = "service_account"
}

variable "log_stream_type" {
  description = "GCS log stream type"
  type        = string
  default     = "Auto"
}

variable "log_types" {
  description = "Log types for the default prefix mapping"
  type        = list(string)
  default     = ["GCP.AuditLog"]
}

# --- Alarm variables ---

variable "alarm_type" {
  description = "Alarm type. Only SOURCE_NO_DATA is currently supported by the API."
  type        = string
  default     = "SOURCE_NO_DATA"
}

variable "minutes_threshold" {
  description = "No-data evaluation period (minutes). Must be between 15 and 43200 (30 days)."
  type        = number
  default     = 60
}
