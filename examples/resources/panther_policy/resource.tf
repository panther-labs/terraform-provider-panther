# Manage cloud security policy for resource compliance
resource "panther_policy" "example" {
  display_name = "S3 Bucket Encryption Policy"
  body         = <<-EOT
    def policy(resource):
        # Check if S3 bucket has encryption enabled
        encryption = resource.get('EncryptionConfiguration', {})
        rules = encryption.get('Rules', [])
        return len(rules) > 0
  EOT
  severity     = "MEDIUM"
  description  = "Ensures S3 buckets have encryption enabled"
  enabled      = true

  resource_types = [
    "AWS.S3.Bucket"
  ]

  tags = [
    "compliance",
    "encryption"
  ]
}
