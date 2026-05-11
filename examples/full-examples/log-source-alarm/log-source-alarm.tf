resource "panther_httpsource" "parent" {
  integration_label = var.integration_label
  log_stream_type   = var.log_stream_type
  log_types         = var.log_types
  auth_method       = var.auth_method
}

resource "panther_log_source_alarm" "example" {
  source_id         = panther_httpsource.parent.id
  type              = var.alarm_type
  minutes_threshold = var.minutes_threshold
}
