# Manage an S3 Log Source integration in Panther.
resource "panther_s3_source" "example" {
  aws_account_id                               = "123456789012"
  name                                         = "my-s3-logs"
  log_processing_role_arn                      = "arn:aws:iam::123456789012:role/PantherLogProcessingRole"
  log_stream_type                              = "Auto"
  panther_managed_bucket_notifications_enabled = true
  bucket_name                                  = "my-log-bucket"
  prefix_log_types = [{
    excluded_prefixes = []
    log_types         = ["AWS.CloudTrail"]
    prefix            = ""
  }]
}