resource "panther_httpsource" "example_http_source" {
  integration_label = var.integration_label
  log_stream_type   = var.log_stream_type
  log_types         = var.log_types
  auth_method       = var.auth_method
  auth_header_key   = var.auth_header_key
  auth_secret_value = var.auth_secret_value
}
