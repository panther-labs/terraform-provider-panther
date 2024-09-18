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
