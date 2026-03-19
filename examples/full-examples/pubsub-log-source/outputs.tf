output "log-source-id" {
  description = "Pub/Sub Log Source id"
  value       = panther_pubsubsource.example_pubsub_source.id
}

output "credentials-type" {
  description = "Credentials type (service_account or wif)"
  value       = panther_pubsubsource.example_pubsub_source.credentials_type
}
