# Log source that the alarm attaches to. In a real configuration this can be any
# panther_s3_source, panther_httpsource, panther_pubsubsource, or panther_gcssource.
resource "panther_httpsource" "example" {
  integration_label = "example-http-source"
  log_stream_type   = "JSON"
  log_types         = ["AWS.CloudTrail"]
  auth_method       = "SharedSecret"
  auth_header_key   = "x-api-key"
  auth_secret_value = "change-me"
}

resource "panther_log_source_alarm" "example" {
  source_id         = panther_httpsource.example.id
  type              = "SOURCE_NO_DATA"
  minutes_threshold = 60
}