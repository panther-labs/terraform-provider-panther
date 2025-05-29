# Manage Http Log Source integration
resource "panther_httpsource" "example_http_source" {
  integration_label = "test-http-source"
  log_stream_type   = "JSON"
  log_types         = ["AWS.CloudFrontAccess"]
  auth_method       = "SharedSecret"
  auth_header_key   = "x-api-key"
  auth_secret_value = "secret"
  auth_username     = ""
  auth_password     = ""
  auth_hmac_alg     = ""
  auth_bearer_token = ""
  log_stream_type_options = {
    json_array_envelope_field = "records"
  }
}
