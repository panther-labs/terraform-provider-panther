# Manage a GCS (Google Cloud Storage) Log Source integration in Panther.
# GCS sources ingest logs from a GCS bucket via Pub/Sub notifications.
resource "panther_gcssource" "example" {
  integration_label = "my-gcs-logs"
  subscription_id   = "my-gcs-notification-subscription"
  gcs_bucket        = "my-log-bucket"
  project_id        = "my-gcp-project"
  credentials       = file("gcp-credentials.json")
  credentials_type  = "service_account"
  log_stream_type   = "Auto"

  prefix_log_types = [{
    prefix            = ""
    log_types         = ["GCP.AuditLog"]
    excluded_prefixes = []
  }]
}
