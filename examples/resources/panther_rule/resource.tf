# Manage detection rule for log analysis
resource "panther_rule" "example" {
  display_name         = ""
  body                 = ""
  severity             = ""
  description          = ""
  enabled              = true
  dedup_period_minutes = 60
  log_types = [
    ""
  ]
  tags = [
    ""
  ]
  runbook = ""
}