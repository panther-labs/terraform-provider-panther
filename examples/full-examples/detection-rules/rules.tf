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