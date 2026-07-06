# Manage scheduled detection rule for query results
resource "panther_scheduled_rule" "example" {
  display_name         = "High Volume Failed Logins"
  body                 = <<-EOT
    def rule(event):
        # Check if query results exceed threshold
        failed_count = event.get('failed_login_count', 0)
        return failed_count > 10
  EOT
  severity             = "HIGH"
  description          = "Detects high volume of failed login attempts from scheduled query"
  enabled              = true
  dedup_period_minutes = 60
  threshold            = 1

  scheduled_queries = [
    "failed-login-aggregation-query"
  ]

  tags = [
    "authentication",
    "security"
  ]

  runbook = "Investigate the source IPs and user accounts for potential brute force attacks"
}
