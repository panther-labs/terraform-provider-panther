/*
Copyright 2023 Panther Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"terraform-provider-panther/internal/client"
)

func TestS3SourceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testS3SourceResourceConfig("test-source"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "aws_account_id", "111122223333"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "name", "test-source"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_processing_role_arn", "arn:aws:iam::111122223333:role/TestRole"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "Lines"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "panther_managed_bucket_notifications_enabled", "true"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "bucket_name", "test_bucket"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", "arn:aws:kms:us-east-1:111122223333:key/testing"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.prefix", "test/prefix"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.excluded_prefixes.0", "test/prefix/excluded"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.log_types.0", "AWS.CloudTrail"),
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
				Config: providerConfig + testUpdatedS3SourceResourceConfig("test-source-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "aws_account_id", "111122223333"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "name", "test-source-updated"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_processing_role_arn", "arn:aws:iam::111122223333:role/TestRole"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "JSON"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type_options.json_array_envelope_field", "records"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type_options.xml_root_element", "root"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "panther_managed_bucket_notifications_enabled", "true"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "bucket_name", "test_bucket"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", "arn:aws:kms:us-east-1:111122223333:key/testing"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.prefix", "test/prefix"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.excluded_prefixes.0", "test/prefix/excluded"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.log_types.0", "AWS.CloudTrail"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestPrefixLogTypesToInput(t *testing.T) {
	prefixLogTypes := []PrefixLogTypesModel{{
		ExcludedPrefixes: []types.String{
			types.StringValue("test/prefix/excluded"),
			types.StringValue("test/prefix/excluded2")},
		LogTypes: []types.String{
			types.StringValue("AWS.CloudTrail"),
			types.StringValue("AWS.ALB")},
		Prefix: types.StringValue("test/prefix"),
	}}
	input := prefixLogTypesToInput(prefixLogTypes)
	assert.Len(t, input, 1)

	// excluded prefixes
	assert.Len(t, input[0].ExcludedPrefixes, 2)
	assert.Contains(t, input[0].ExcludedPrefixes, "test/prefix/excluded")
	assert.Contains(t, input[0].ExcludedPrefixes, "test/prefix/excluded2")

	// log types
	assert.Len(t, input[0].LogTypes, 2)
	assert.Contains(t, input[0].LogTypes, "AWS.CloudTrail")
	assert.Contains(t, input[0].LogTypes, "AWS.ALB")

	// prefix
	assert.Equal(t, "test/prefix", input[0].Prefix)
}

func TestPrefixLogTypesToModel(t *testing.T) {
	prefixLogTypes := []client.S3PrefixLogTypes{{
		ExcludedPrefixes: []string{"test/prefix/excluded", "test/prefix/excluded2"},
		LogTypes:         []string{"AWS.CloudTrail", "AWS.ALB"},
		Prefix:           "test/prefix",
	}}
	input := prefixLogTypesToModel(prefixLogTypes)
	assert.Len(t, input, 1)

	// excluded prefixes
	assert.Len(t, input[0].ExcludedPrefixes, 2)
	assert.Contains(t, input[0].ExcludedPrefixes, types.StringValue("test/prefix/excluded"))
	assert.Contains(t, input[0].ExcludedPrefixes, types.StringValue("test/prefix/excluded2"))

	// log types
	assert.Len(t, input[0].LogTypes, 2)
	assert.Contains(t, input[0].LogTypes, types.StringValue("AWS.CloudTrail"))
	assert.Contains(t, input[0].LogTypes, types.StringValue("AWS.ALB"))

	// prefix
	assert.Equal(t, types.StringValue("test/prefix"), input[0].Prefix)
}

func testS3SourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id = "111122223333"
  name = "%v"
  log_processing_role_arn = "arn:aws:iam::111122223333:role/TestRole"
  log_stream_type = "Lines"
  panther_managed_bucket_notifications_enabled = true
  bucket_name = "test_bucket"
  kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/testing"
  prefix_log_types = [{
    excluded_prefixes = ["test/prefix/excluded"]
    log_types         = ["AWS.CloudTrail"]
    prefix            = "test/prefix"
  }]
}
`, name)
}

func testUpdatedS3SourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id = "111122223333"
  name = "%v"
  log_processing_role_arn = "arn:aws:iam::111122223333:role/TestRole"
  log_stream_type = "JSON"
  log_stream_type_options = {
    json_array_envelope_field = "records"
    xml_root_element = "root"
  }
  panther_managed_bucket_notifications_enabled = true
  bucket_name = "test_bucket"
  kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/testing"
  prefix_log_types = [{
    excluded_prefixes = ["test/prefix/excluded"]
    log_types         = ["AWS.CloudTrail"]
    prefix            = "test/prefix"
  }]
}
`, name)
}
