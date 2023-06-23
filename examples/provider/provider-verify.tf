terraform {
  required_providers {
    panther = {
      source = "panther.com/dev/panther"
    }
  }
}

provider "panther" {
  token = "11aKniRUNROY7ttQzRnL3DRUviS7Imy84ueUboV4"
  url = "https://api.snowflake.staging.runpanther.xyz/public/graphql"
}

resource "panther_s3_source" "test_source" {
  aws_account_id = "test"
  name = "test"
  log_processing_role_arn = "test"
  log_stream_type = "Lines"
#  is_managed_bucket_notifications_enabled = false
  kms_key = "test"
  bucket_name = "test"

}
