resource "panther_httpsource" "example_http_source" {
  integration_label     = var.integration_label
  log_stream_type       = var.log_stream_type
  log_types             = var.log_types
  security_type         = var.security_type
  security_header_key   = var.security_header_key
  security_secret_value = var.security_secret_value
  security_username     = var.security_username
  security_password     = var.security_password
  security_alg          = var.security_alg
}