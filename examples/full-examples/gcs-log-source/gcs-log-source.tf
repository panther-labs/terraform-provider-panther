resource "panther_gcssource" "example_gcs_source" {
  integration_label = var.integration_label
  subscription_id   = var.subscription_id
  project_id        = var.project_id
  gcs_bucket        = var.gcs_bucket
  credentials       = file(var.credentials_file)
  credentials_type  = var.credentials_type
  log_stream_type   = var.log_stream_type
  log_stream_type_options = {
    json_array_envelope_field = var.json_array_envelope_field
  }

  prefix_log_types = var.prefix_log_types
}
