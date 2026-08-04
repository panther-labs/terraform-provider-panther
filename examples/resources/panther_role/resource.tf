# Manage a Panther role
resource "panther_role" "example_role" {
  name = "example-role"
  permissions = [
    "AlertRead",
    "RuleRead",
  ]
  log_type_access_kind = "ALLOW_ALL"
}
