resource "panther_role" "example" {
  name = "LogAnalyst"

  permissions = [
    "LogAnalysis:ReadData",
    "LogAnalysis:ViewLogSources",
  ]
}
