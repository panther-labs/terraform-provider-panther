# Manage GCP Pub/Sub Log Source integration
resource "panther_pubsubsource" "example_pubsub_source" {
  integration_label = "my-gcp-logs"
  subscription_id   = "my-subscription"
  project_id        = "my-gcp-project"
  credentials       = file("gcp-credentials.json")
  credentials_type  = "service_account"
  log_types         = ["GCP.AuditLog"]
  log_stream_type   = "Auto"
}
