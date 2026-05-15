output "aws_cloud_account_id" {
  description = "The Panther integration ID of the AWS Cloud Account"
  value       = panther_aws_cloud_account.example.id
}

output "aws_cloud_account_label" {
  description = "The integration label of the AWS Cloud Account"
  value       = panther_aws_cloud_account.example.integration_label
}
