output "log-source-name" {
  description = "http Log Source name"
  value       = panther_httpsource.example_http_source.integration_label
}
