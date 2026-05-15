variable "token" {
  description = "Panther API token"
  type        = string
}

variable "url" {
  description = "Panther API URL"
  type        = string
}

variable "integration_label" {
  description = "Display name for the AWS Cloud Account integration (alphanumeric, dashes, spaces; max 36 chars)"
  type        = string
  default     = "provider-aws-cloud-account-test"
}

variable "aws_account_id" {
  description = "12-digit AWS account ID where the audit role is deployed"
  type        = string
}

variable "audit_role" {
  description = "IAM role ARN that Panther assumes to scan the AWS account"
  type        = string
}

variable "region_ignore_list" {
  description = "AWS regions to exclude from scanning"
  type        = list(string)
  default     = []
}

variable "resource_type_ignore_list" {
  description = "AWS resource types to exclude from scanning (e.g. AWS.S3.Bucket)"
  type        = list(string)
  default     = []
}

variable "resource_regex_ignore_list" {
  description = "Regex patterns matching resource ARNs to exclude from scanning"
  type        = list(string)
  default     = []
}
