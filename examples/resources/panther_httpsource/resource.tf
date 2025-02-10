# Manage Http Log Source integration
resource "panther_httpsource" "example_http_source" {
  integration_label = ""
  log_stream_type   = "JSON"
  log_types         = ""
  auth_method       = "SharedSecret"
  auth_header_key   = ""
  auth_secret_value = ""
  auth_username     = ""
  auth_password     = ""
  auth_hmac_alg     = ""
  auth_bearer_token = ""
}
