# Manage S3 Log Source integration
resource "panther_s3_source" "test_source" {
  aws_account_id                               = ""
  name                                         = ""
  log_processing_role_arn                      = ""
  log_stream_type                              = "Lines"
  panther_managed_bucket_notifications_enabled = true
  kms_key_arn                                  = ""
  bucket_name                                  = ""
  prefix_log_types = [{
    excluded_prefixes = []
    log_types         = []
    prefix            = ""
  }]
}