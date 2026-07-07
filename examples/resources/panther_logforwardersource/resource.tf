# Manage a Panther Log Forwarder (PLF) Log Source integration
resource "panther_logforwardersource" "example" {
  integration_label = "my-log-forwarder"
  log_stream_type   = "JSON"
  log_types         = ["AWS.CloudTrail"]
}
