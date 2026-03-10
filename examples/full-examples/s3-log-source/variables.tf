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

variable "log_processing_role_arn" {
  description = "Role ARN of the configured role for accessing the bucket"
  type        = string
}

variable "bucket_name" {
  default     = "test-bucket"
  description = "Bucket name"
  type        = string
}

variable "log_stream_type" {
  description = "The type of log stream (e.g. Lines, JsonArray, CloudWatchLogs, XML)"
  type        = string
}

variable "kms_key_arn" {
  description = "ARN of the KMS key used to encrypt the S3 bucket"
  type        = string
}

variable "log_types" {
  description = "List of log types for the prefix"
  type        = list(string)
}

variable "prefix" {
  description = "S3 prefix to filter logs"
  type        = string
}

variable "json_array_envelope_field" {
  description = "Path to the array value to extract elements from, only applicable if logStreamType is JsonArray. Leave empty if the input JSON is an array itself"
  type        = string
}

variable "retain_envelope_fields" {
  description = "When enabled, envelope metadata from CloudWatch Logs is preserved in a p_header column on each unpacked event."
  type        = bool
  default     = false
}

variable "xml_root_element" {
  description = "The root element name for XML streams, only applicable if logStreamType is XML. Leave empty if the XML events are not enclosed in a root element"
  type        = string
}
