# Import an existing AWS Cloud Account integration by its Panther integration ID.
# The integration ID is a UUID v4 — read it from the Panther Console URL when
# viewing the cloud account (Configure > Cloud Accounts). A REST list endpoint
# is not yet available.
#
# Caveat: if the integration was created with a non-default scan interval via
# the Panther UI or GraphQL, the first `terraform apply` after import will
# silently reset the scan interval to 24 hours (1440 minutes). The REST API
# does not currently expose `scanIntervalMins`.
terraform import panther_aws_cloud_account.example <integration-id>
