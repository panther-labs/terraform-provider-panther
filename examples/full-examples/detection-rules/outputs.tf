output "console_login_rule_id" {
  description = "ID of the console login detection rule"
  value       = panther_rule.console_login.id
}

output "failed_login_rule_id" {
  description = "ID of the failed login detection rule"
  value       = panther_rule.failed_login.id
}