resource "panther_gcssource" "parent" {
  integration_label = var.integration_label
  subscription_id   = var.subscription_id
  project_id        = var.project_id
  gcs_bucket        = var.gcs_bucket
  credentials       = file(var.credentials_file)
  credentials_type  = var.credentials_type
  log_stream_type   = var.log_stream_type

  prefix_log_types = [{
    prefix            = ""
    log_types         = var.log_types
    excluded_prefixes = []
  }]
}

resource "panther_log_source_alarm" "example" {
  source_id         = panther_gcssource.parent.id
  type              = var.alarm_type
  minutes_threshold = var.minutes_threshold
}
