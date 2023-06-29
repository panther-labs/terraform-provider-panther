// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"terraform-provider-panther/internal/client"
	"terraform-provider-panther/internal/client/clientfakes"
)

func TestS3SourceResource(t *testing.T) {
	mockClient := clientfakes.FakeClient{}
	logStreamType := "Lines"
	logProcessingRole := "arn:aws:iam::111122223333:role/TestRole"
	mockClient.CreateS3SourceReturns(client.CreateS3SourceOutput{
		LogSource: &client.S3LogIntegration{
			AwsAccountID:               "111122223333",
			IntegrationLabel:           "test-source",
			IntegrationID:              "test-id",
			LogStreamType:              &logStreamType,
			ManagedBucketNotifications: true,
			S3Bucket:                   "test_bucket",
			LogProcessingRole:          &logProcessingRole,
		},
	}, nil)
	mockClient.UpdateS3SourceReturns(client.UpdateS3SourceOutput{
		LogSource: &client.S3LogIntegration{
			AwsAccountID:               "111122223333",
			IntegrationLabel:           "test-source-updated",
			IntegrationID:              "test-id",
			LogStreamType:              &logStreamType,
			ManagedBucketNotifications: true,
			S3Bucket:                   "test_bucket",
			LogProcessingRole:          &logProcessingRole,
		},
	}, nil)
	mockClient.GetS3SourceReturnsOnCall(0, &client.S3LogIntegration{
		AwsAccountID:               "111122223333",
		IntegrationLabel:           "test-source",
		IntegrationID:              "test-id",
		LogStreamType:              &logStreamType,
		ManagedBucketNotifications: true,
		S3Bucket:                   "test_bucket",
		LogProcessingRole:          &logProcessingRole,
	}, nil)
	mockClient.GetS3SourceReturnsOnCall(1, &client.S3LogIntegration{
		AwsAccountID:               "111122223333",
		IntegrationLabel:           "test-source",
		IntegrationID:              "test-id",
		LogStreamType:              &logStreamType,
		ManagedBucketNotifications: true,
		S3Bucket:                   "test_bucket",
		LogProcessingRole:          &logProcessingRole,
	}, nil)
	mockClient.GetS3SourceReturnsOnCall(2, &client.S3LogIntegration{
		AwsAccountID:               "111122223333",
		IntegrationLabel:           "test-source-updated",
		IntegrationID:              "test-id",
		LogStreamType:              &logStreamType,
		ManagedBucketNotifications: true,
		S3Bucket:                   "test_bucket",
		LogProcessingRole:          &logProcessingRole,
	}, nil)
	mockClient.GetS3SourceReturnsOnCall(3, &client.S3LogIntegration{
		AwsAccountID:               "111122223333",
		IntegrationLabel:           "test-source-updated",
		IntegrationID:              "test-id",
		LogStreamType:              &logStreamType,
		ManagedBucketNotifications: true,
		S3Bucket:                   "test_bucket",
		LogProcessingRole:          &logProcessingRole,
	}, nil)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: newTestAccProtoV6ProviderFactories(mockClient),
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testS3SourceResourceConfig("test-source"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "aws_account_id", "111122223333"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "name", "test-source"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_processing_role_arn", "arn:aws:iam::111122223333:role/TestRole"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "Lines"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "is_managed_bucket_notifications_enabled", "true"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "bucket_name", "test_bucket"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_s3_source.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testS3SourceResourceConfig("test-source-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "name", "test-source-updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testS3SourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id = "111122223333"
  name = "%v"
  log_processing_role_arn = "arn:aws:iam::111122223333:role/TestRole"
  log_stream_type = "Lines"
  is_managed_bucket_notifications_enabled = true
#  kms_key = "test"
  bucket_name = "test_bucket"
  prefix_log_types = []
}
`, name)
}
