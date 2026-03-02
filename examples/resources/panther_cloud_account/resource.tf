# Manage AWS Cloud Account integration
resource "panther_cloud_account" "example" {
  aws_account_id = ""
  label          = ""
  audit_role     = ""
  aws_region_ignore_list = [
    ""
  ]
  resource_type_ignore_list = [
    ""
  ]
  resource_regex_ignore_list = [
    ""
  ]
}