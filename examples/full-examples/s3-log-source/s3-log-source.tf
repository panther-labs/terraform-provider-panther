resource "panther_s3_source" "example_s3_source" {
  aws_account_id                               = var.aws_account_id
  name                                         = "provider-log-source-test"
  log_processing_role_arn                      = var.log_processing_role_arn
  log_stream_type                              = "Lines"
  panther_managed_bucket_notifications_enabled = true
  kms_key_arn                                  = ""
  bucket_name                                  = var.bucket_name
  prefix_log_types = [{
    excluded_prefixes = []
    log_types         = []
    prefix            = ""
  }]
}