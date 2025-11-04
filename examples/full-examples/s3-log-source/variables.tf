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

variable "json_array_envelope_field" {
  description = "Path to the array value to extract elements from, only applicable if logStreamType is JsonArray. Leave empty if the input JSON is an array itself"
  type        = string
}

variable "xml_root_element" {
  description = "The root element name for XML streams, only applicable if logStreamType is XML. Leave empty if the XML events are not enclosed in a root element"
  type        = string
}
