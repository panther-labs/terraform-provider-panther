# Manage simple detection rule with YAML-based detection
resource "panther_simple_rule" "example" {
  display_name         = "AWS Console Login Detection"
  detection            = <<-EOT
    MatchFilters:
      - Key: eventName
        Condition: Equals
        Values:
          - ConsoleLogin
      - Key: userIdentity.type
        Condition: Equals
        Values:
          - IAMUser
  EOT
  severity             = "CRITICAL"
  description          = "Detects AWS console login events from IAM users"
  enabled              = true
  dedup_period_minutes = 60
  threshold            = 1

  log_types = [
    "AWS.CloudTrail"
  ]

  tags = [
    "authentication",
    "aws"
  ]

  alert_title   = "AWS Console Login: {{p_any_aws_account_ids}}"
  alert_context = <<-EOT
    User: {{userIdentity.userName}}
    Source IP: {{sourceIPAddress}}
  EOT

  runbook = "Verify the login is legitimate and investigate if from unexpected location"
}
