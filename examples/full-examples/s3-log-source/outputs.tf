output "s3_source_id" {
  description = "The ID of the S3 Log Source integration"
  value       = panther_s3_source.example_s3_source.id
}

output "s3_source_name" {
  description = "S3 Log Source name"
  value       = panther_s3_source.example_s3_source.name
}
