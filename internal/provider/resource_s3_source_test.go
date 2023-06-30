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
	// set up panther graphQL client mocks
	mockClient := initMockClient()
	// run tests
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
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", "key"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.prefix", "prefix"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.excluded_prefixes.0", "excluded-prefix"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.log_types.0", "AWS.Audit"),
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

func initMockClient() client.Client {
	mockClient := clientfakes.FakeClient{}
	logStreamType := "Lines"
	logProcessingRole := "arn:aws:iam::111122223333:role/TestRole"
	kmsKey := "key"
	originalSource := client.S3LogIntegration{
		AwsAccountID:               "111122223333",
		IntegrationLabel:           "test-source",
		IntegrationID:              "test-id",
		LogStreamType:              &logStreamType,
		ManagedBucketNotifications: true,
		S3Bucket:                   "test_bucket",
		LogProcessingRole:          &logProcessingRole,
		KmsKey:                     &kmsKey,
		S3PrefixLogTypes: []client.S3PrefixLogTypes{{
			Prefix:           "prefix",
			LogTypes:         []string{"AWS.Audit"},
			ExcludedPrefixes: []string{"excluded-prefix"},
		}},
	}
	updatedSource := originalSource
	updatedSource.IntegrationLabel = "test-source-updated"
	mockClient.CreateS3SourceReturns(client.CreateS3SourceOutput{
		LogSource: &originalSource,
	}, nil)
	mockClient.UpdateS3SourceReturns(client.UpdateS3SourceOutput{
		LogSource: &updatedSource,
	}, nil)
	mockClient.GetS3SourceReturnsOnCall(0, &originalSource, nil)
	mockClient.GetS3SourceReturnsOnCall(1, &originalSource, nil)
	mockClient.GetS3SourceReturnsOnCall(2, &updatedSource, nil)
	mockClient.GetS3SourceReturnsOnCall(3, &updatedSource, nil)
	return &mockClient
}

func testS3SourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id = "111122223333"
  name = "%v"
  log_processing_role_arn = "arn:aws:iam::111122223333:role/TestRole"
  log_stream_type = "Lines"
  is_managed_bucket_notifications_enabled = true
  bucket_name = "test_bucket"
  kms_key_arn = "key"
  prefix_log_types = [{
    excluded_prefixes = ["excluded-prefix"]
    log_types         = ["AWS.Audit"]
    prefix            = "prefix"
  }]
}
`, name)
}
