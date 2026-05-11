variable "token" {
  description = "Panther API token"
  type        = string
  sensitive   = true
}

variable "url" {
  description = "Panther API URL"
  type        = string
}

# --- HTTP source (parent) variables ---

variable "integration_label" {
  description = "Display name for the HTTP log source integration"
  type        = string
  default     = "provider-log-source-alarm-test"
}

variable "log_stream_type" {
  description = "HTTP log stream type. One of: Auto, JSON, JsonArray, Lines, XML."
  type        = string
  default     = "Auto"
}

variable "log_types" {
  description = "Log types accepted by the HTTP source"
  type        = list(string)
  default     = ["AWS.CloudFrontAccess"]
}

variable "auth_method" {
  description = "HTTP source authentication method. One of: SharedSecret, HMAC, Bearer, Basic, None."
  type        = string
  default     = "None"
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
