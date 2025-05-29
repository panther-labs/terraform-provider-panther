# Manage S3 Log Source integration
resource "panther_s3_source" "test_source" {
  aws_account_id                               = "123456789012"
  name                                         = "test-s3-source"
  log_processing_role_arn                      = "arn:aws:iam::123456789012:role/panther-s3-log-processing-role"
  log_stream_type                              = "Lines"
  panther_managed_bucket_notifications_enabled = true
  kms_key_arn                                  = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
  bucket_name                                  = "test-bucket"
  prefix_log_types = [{
    excluded_prefixes = []
    log_types         = ["AWS.CloudFrontAccess"]
    prefix            = "test-prefix"
  }]
}