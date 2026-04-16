variable "token" {
  description = "Panther API token"
  type        = string
}

variable "url" {
  description = "Panther API URL"
  type        = string
}

variable "aws_account_id" {
  description = "AWS Account ID where the bucket is located"
  type        = string
}

variable "name" {
  description = "Display name for the S3 Log Source integration"
  type        = string
  default     = "provider-log-source-test"
}

variable "log_processing_role_arn" {
  description = "Role ARN of the configured role for accessing the bucket"
  type        = string
}

variable "bucket_name" {
  description = "Bucket name"
  type        = string
}

variable "log_stream_type" {
  description = "The type of log stream: Auto, Lines, JSON, JsonArray, CloudWatchLogs, or XML"
  type        = string
  default     = "Auto"
}

variable "kms_key_arn" {
  description = "ARN of the KMS key used to encrypt the S3 bucket (leave empty if not using KMS)"
  type        = string
  default     = ""
}

variable "panther_managed_bucket_notifications_enabled" {
  description = "True if bucket notifications are being managed by Panther"
  type        = bool
  default     = true
}

variable "log_types" {
  description = "List of log types for the prefix"
  type        = list(string)
}

variable "prefix" {
  description = "S3 prefix to filter logs (leave empty to match all objects)"
  type        = string
  default     = ""
}

# These variables are only needed when log_stream_type_options is uncommented in s3-log-source.tf.

variable "json_array_envelope_field" {
  description = "Path to the array value to extract elements from. Only applicable when log_stream_type is JsonArray."
  type        = string
  default     = ""
}

variable "retain_envelope_fields" {
  description = "When enabled, envelope metadata from CloudWatch Logs is preserved in a p_header column on each unpacked event."
  type        = bool
  default     = false
}

variable "xml_root_element" {
  description = "The root element name for XML streams. Only applicable when log_stream_type is XML."
  type        = string
  default     = ""
}
