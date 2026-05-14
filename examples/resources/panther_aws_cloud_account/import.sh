# Import an existing AWS Cloud Account integration by its Panther integration ID.
# The integration ID is a UUID v4 — read it from the Panther Console URL when
# viewing the cloud account (Configure > Cloud Accounts). A REST list endpoint
# is not yet available.
terraform import panther_aws_cloud_account.example 12345678-1234-1234-1234-123456789012
