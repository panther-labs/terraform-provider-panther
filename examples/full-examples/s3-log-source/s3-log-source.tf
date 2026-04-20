resource "panther_s3_source" "example_s3_source" {
  aws_account_id                               = var.aws_account_id
  name                                         = var.name
  log_processing_role_arn                      = var.log_processing_role_arn
  log_stream_type                              = var.log_stream_type
  panther_managed_bucket_notifications_enabled = var.panther_managed_bucket_notifications_enabled
  kms_key_arn                                  = var.kms_key_arn
  bucket_name                                  = var.bucket_name
  prefix_log_types                             = var.prefix_log_types

  # Uncomment when using JsonArray, CloudWatchLogs, or XML log stream types:
  #
  # log_stream_type_options = {
  #   json_array_envelope_field = var.json_array_envelope_field   # JsonArray only
  #   retain_envelope_fields    = var.retain_envelope_fields      # CloudWatchLogs only
  #   xml_root_element          = var.xml_root_element            # XML only
  # }
}
