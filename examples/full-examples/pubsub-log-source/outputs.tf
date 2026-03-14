output "log-source-id" {
  description = "Pub/Sub Log Source id"
  value       = panther_pubsubsource.example_pubsub_source.id
}

output "credentials-type" {
  description = "Derived credentials type (service_account or external_account)"
  value       = panther_pubsubsource.example_pubsub_source.credentials_type
}
