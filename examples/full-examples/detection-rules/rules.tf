# Standard detection rule for log analysis
resource "panther_rule" "console_login" {
  display_name = "AWS Console Login Detection"
  body         = <<-EOT
    def rule(event):
        return event.get('eventName') == 'ConsoleLogin'
  EOT
  severity     = "MEDIUM"
  enabled      = true

  log_types = [
    "AWS.CloudTrail"
  ]

  tags = [
    "authentication",
    "aws"
  ]

  description = "Detects AWS console login events"
}

resource "panther_rule" "failed_login" {
  display_name = "Failed Console Login"
  body         = <<-EOT
    def rule(event):
        return (event.get('eventName') == 'ConsoleLogin' and
                event.get('errorCode') == 'SigninFailure')
  EOT
  severity     = "HIGH"
  enabled      = true

  log_types = [
    "AWS.CloudTrail"
  ]

  tags = [
    "authentication",
    "security"
  ]

  description = "Detects failed AWS console login attempts"
  runbook     = "Investigate the source IP and user account for potential compromise"
}

# Simple rule with YAML-based detection
resource "panther_simple_rule" "root_login" {
  display_name         = "Root Account Login"
  detection            = <<-EOT
    MatchFilters:
      - Key: eventName
        Condition: Equals
        Values:
          - ConsoleLogin
      - Key: userIdentity.type
        Condition: Equals
        Values:
          - Root
  EOT
  severity             = "CRITICAL"
  description          = "Detects AWS root account console login"
  enabled              = true
  dedup_period_minutes = 60
  threshold            = 1

  log_types = [
    "AWS.CloudTrail"
  ]

  tags = [
    "authentication",
    "critical"
  ]

  alert_title   = "Root Login: {{p_any_aws_account_ids}}"
  alert_context = "Source IP: {{sourceIPAddress}}"
  runbook       = "Immediately verify root account login and enable MFA if not enabled"
}

# Scheduled rule for query result analysis
resource "panther_scheduled_rule" "brute_force_detection" {
  display_name         = "Brute Force Attack Detection"
  body                 = <<-EOT
    def rule(event):
        # Query aggregates failed login attempts per IP
        failed_count = event.get('failed_login_count', 0)
        unique_users = event.get('unique_user_count', 0)
        return failed_count > 50 and unique_users > 3
  EOT
  severity             = "HIGH"
  description          = "Detects potential brute force attacks from scheduled query results"
  enabled              = true
  dedup_period_minutes = 120
  threshold            = 1

  scheduled_queries = [
    "failed-login-aggregation"
  ]

  tags = [
    "authentication",
    "security"
  ]

  runbook = "Block source IPs showing brute force patterns and investigate affected accounts"
}

# Policy for cloud resource compliance
resource "panther_policy" "s3_encryption" {
  display_name = "S3 Bucket Encryption Required"
  body         = <<-EOT
    def policy(resource):
        encryption = resource.get('EncryptionConfiguration', {})
        rules = encryption.get('Rules', [])
        return len(rules) > 0 and any(
            rule.get('ApplyServerSideEncryptionByDefault', {}).get('SSEAlgorithm')
            for rule in rules
        )
  EOT
  severity     = "HIGH"
  enabled      = true

  resource_types = [
    "AWS.S3.Bucket"
  ]

  tags = [
    "compliance",
    "encryption"
  ]

  description = "Ensures all S3 buckets have encryption enabled"
}