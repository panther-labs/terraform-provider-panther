# Manage Http Log Source integration
resource "panther_httpsource" "example_http_source" {
  integration_label     = ""
  log_stream_type       = "JSON"
  log_types             = ""
  security_type         = "SharedSecret"
  security_header_key   = ""
  security_secret_value = ""
  security_username     = ""
  security_password     = ""
  security_alg          = ""
}
