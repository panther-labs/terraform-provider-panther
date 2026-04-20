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
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"terraform-provider-panther/internal/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

// s3TestConfig holds environment-specific values for S3 acceptance tests.
// Loaded from PANTHER_S3_* env vars — see .env.s3.test.
type s3TestConfig struct {
	awsAccountID         string
	bucketName           string
	logProcessingRoleARN string
	kmsKeyARN            string
}

// loadS3TestConfig reads S3-specific env vars. Returns ok=false if required vars are missing.
func loadS3TestConfig(t *testing.T) (cfg s3TestConfig, ok bool) {
	t.Helper()
	cfg.awsAccountID = os.Getenv("PANTHER_S3_AWS_ACCOUNT_ID")
	cfg.bucketName = os.Getenv("PANTHER_S3_BUCKET_NAME")
	cfg.logProcessingRoleARN = os.Getenv("PANTHER_S3_LOG_PROCESSING_ROLE_ARN")
	cfg.kmsKeyARN = os.Getenv("PANTHER_S3_KMS_KEY_ARN") // optional

	if cfg.awsAccountID == "" || cfg.bucketName == "" || cfg.logProcessingRoleARN == "" {
		return cfg, false
	}
	return cfg, true
}

func TestS3SourceResource(t *testing.T) {
	cfg, ok := loadS3TestConfig(t)
	if !ok {
		t.Skip("Skipping: PANTHER_S3_AWS_ACCOUNT_ID, PANTHER_S3_BUCKET_NAME, and PANTHER_S3_LOG_PROCESSING_ROLE_ARN must be set")
	}

	name := strings.ReplaceAll(uuid.NewString(), "-", "")
	nameUpdated := strings.ReplaceAll(uuid.NewString(), "-", "")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkS3SourceDestroyed,
		Steps: []resource.TestStep{
			// Step 1: Create basic S3 source.
			// The framework automatically runs a post-apply plan to verify no perpetual diffs.
			{
				Config: providerConfig + testS3SourceConfig_Basic(cfg, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "aws_account_id", cfg.awsAccountID),
					resource.TestCheckResourceAttr("panther_s3_source.test", "name", name),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_processing_role_arn", cfg.logProcessingRoleARN),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "Lines"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "panther_managed_bucket_notifications_enabled", "true"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "bucket_name", cfg.bucketName),
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", cfg.kmsKeyARN),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.prefix", "test/prefix"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.excluded_prefixes.0", "test/prefix/excluded"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.log_types.0", "AWS.CloudTrail"),
					resource.TestCheckResourceAttrSet("panther_s3_source.test", "id"),
				),
			},
			// Step 2: Import by ID. The framework verifies a subsequent plan shows no diff.
			{
				ResourceName:      "panther_s3_source.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update — change name, switch to CloudWatchLogs with retainEnvelopeFields, clear KMS key.
			{
				Config: providerConfig + testS3SourceConfig_CloudWatchLogs(cfg, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "CloudWatchLogs"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type_options.retain_envelope_fields", "true"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", ""),
				),
			},
			// Step 4: Update — multiple prefix_log_types, add KMS key back, toggle managed notifications off.
			{
				Config: providerConfig + testS3SourceConfig_MultiPrefix(cfg, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "Lines"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "panther_managed_bucket_notifications_enabled", "false"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", cfg.kmsKeyARN),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.#", "2"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.prefix", "cloudtrail/"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.log_types.0", "AWS.CloudTrail"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.0.excluded_prefixes.0", "cloudtrail/debug/"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.1.prefix", "vpcflow/"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.1.log_types.0", "AWS.VPCFlow"),
				),
			},
			// Step 5: Update — revert to Auto stream type, remove log_stream_type_options,
			// restore managed notifications.
			{
				Config: providerConfig + testS3SourceConfig_RevertAuto(cfg, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "Auto"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "panther_managed_bucket_notifications_enabled", "true"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", ""),
					resource.TestCheckResourceAttr("panther_s3_source.test", "prefix_log_types.#", "1"),
				),
			},
			// Step 6: Drift detection — manually delete the resource out-of-band, then verify
			// Terraform's Read detects 404 and proposes recreation.
			{
				Config:             providerConfig + testS3SourceConfig_RevertAuto(cfg, nameUpdated),
				Check:              manuallyDeleteS3Source(t),
				ExpectNonEmptyPlan: true,
			},
			// Step 7: Re-apply the same config to recreate the resource drifted away in Step 6.
			// This leaves a live resource for the framework's cleanup to destroy
			{
				Config: providerConfig + testS3SourceConfig_RevertAuto(cfg, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("panther_s3_source.test", "id"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "Auto"),
				),
			},
		},
	})
}

// manuallyDeleteS3Source bypasses Terraform and deletes via the REST API directly,
// simulating out-of-band deletion for drift detection testing.
func manuallyDeleteS3Source(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["panther_s3_source.test"]
		if !ok {
			return fmt.Errorf("not found: panther_s3_source.test")
		}
		if rs.Primary.ID == "" {
			return errors.New("S3 source ID is not set")
		}
		c := client.NewRESTClient(os.Getenv("PANTHER_API_URL"), os.Getenv("PANTHER_API_TOKEN"))
		if err := client.RestDelete(context.Background(), c, s3SourcePath+"/"+rs.Primary.ID); err != nil {
			return fmt.Errorf("could not delete S3 source: %w", err)
		}
		t.Logf("Manually deleted S3 source %s for drift detection test", rs.Primary.ID)
		return nil
	}
}

// checkS3SourceDestroyed verifies that each panther_s3_source tracked in the final
// test state has actually been removed from the Panther API — closes the silent-failure
// window where Delete() returns no diagnostic but the resource still exists remotely.
func checkS3SourceDestroyed(s *terraform.State) error {
	c := client.NewRESTClient(os.Getenv("PANTHER_API_URL"), os.Getenv("PANTHER_API_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "panther_s3_source" {
			continue
		}
		_, err := client.RestDo[client.S3Source](context.Background(), c, http.MethodGet, s3SourcePath+"/"+rs.Primary.ID, nil)
		if err == nil {
			return fmt.Errorf("S3 source %s still exists after destroy", rs.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking S3 source %s: %w", rs.Primary.ID, err)
		}
	}
	return nil
}

// --- Test configs ---

func testS3SourceConfig_Basic(cfg s3TestConfig, name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id                               = %q
  name                                         = %q
  log_processing_role_arn                      = %q
  log_stream_type                              = "Lines"
  panther_managed_bucket_notifications_enabled = true
  bucket_name                                  = %q
  kms_key_arn                                  = %q
  prefix_log_types = [{
    excluded_prefixes = ["test/prefix/excluded"]
    log_types         = ["AWS.CloudTrail"]
    prefix            = "test/prefix"
  }]
}
`, cfg.awsAccountID, name, cfg.logProcessingRoleARN, cfg.bucketName, cfg.kmsKeyARN)
}

func testS3SourceConfig_CloudWatchLogs(cfg s3TestConfig, name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id                               = %q
  name                                         = %q
  log_processing_role_arn                      = %q
  log_stream_type                              = "CloudWatchLogs"
  log_stream_type_options = {
    retain_envelope_fields = true
  }
  panther_managed_bucket_notifications_enabled = true
  bucket_name                                  = %q
  kms_key_arn                                  = ""
  prefix_log_types = [{
    excluded_prefixes = ["test/prefix/excluded"]
    log_types         = ["AWS.CloudTrail"]
    prefix            = "test/prefix"
  }]
}
`, cfg.awsAccountID, name, cfg.logProcessingRoleARN, cfg.bucketName)
}

func testS3SourceConfig_MultiPrefix(cfg s3TestConfig, name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id                               = %q
  name                                         = %q
  log_processing_role_arn                      = %q
  log_stream_type                              = "Lines"
  panther_managed_bucket_notifications_enabled = false
  bucket_name                                  = %q
  kms_key_arn                                  = %q
  prefix_log_types = [
    {
      excluded_prefixes = ["cloudtrail/debug/"]
      log_types         = ["AWS.CloudTrail"]
      prefix            = "cloudtrail/"
    },
    {
      excluded_prefixes = []
      log_types         = ["AWS.VPCFlow"]
      prefix            = "vpcflow/"
    }
  ]
}
`, cfg.awsAccountID, name, cfg.logProcessingRoleARN, cfg.bucketName, cfg.kmsKeyARN)
}

func testS3SourceConfig_RevertAuto(cfg s3TestConfig, name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id                               = %q
  name                                         = %q
  log_processing_role_arn                      = %q
  log_stream_type                              = "Auto"
  panther_managed_bucket_notifications_enabled = true
  bucket_name                                  = %q
  kms_key_arn                                  = ""
  prefix_log_types = [{
    excluded_prefixes = []
    log_types         = ["AWS.CloudTrail"]
    prefix            = ""
  }]
}
`, cfg.awsAccountID, name, cfg.logProcessingRoleARN, cfg.bucketName)
}

// --- Unit tests ---

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
	prefixLogTypes := []client.S3PrefixLogTypesInput{{
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

func TestS3LogStreamTypeOptions(t *testing.T) {
	attrTypes := s3LogStreamTypeOptionAttrTypes

	t.Run("null object returns nil", func(t *testing.T) {
		result := s3LogStreamTypeOptions(types.ObjectNull(attrTypes))
		assert.Nil(t, result)
	})

	t.Run("unknown object returns nil", func(t *testing.T) {
		result := s3LogStreamTypeOptions(types.ObjectUnknown(attrTypes))
		assert.Nil(t, result)
	})

	t.Run("all null fields returns nil", func(t *testing.T) {
		obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
			"json_array_envelope_field": types.StringNull(),
			"retain_envelope_fields":    types.BoolNull(),
			"xml_root_element":          types.StringNull(),
		})
		result := s3LogStreamTypeOptions(obj)
		assert.Nil(t, result)
	})

	t.Run("single field set", func(t *testing.T) {
		obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
			"json_array_envelope_field": types.StringValue("Records"),
			"retain_envelope_fields":    types.BoolNull(),
			"xml_root_element":          types.StringNull(),
		})
		result := s3LogStreamTypeOptions(obj)
		assert.NotNil(t, result)
		assert.Equal(t, "Records", result.JsonArrayEnvelopeField)
		assert.False(t, result.RetainEnvelopeFields)
		assert.Empty(t, result.XmlRootElement)
	})

	t.Run("retain_envelope_fields set", func(t *testing.T) {
		obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
			"json_array_envelope_field": types.StringNull(),
			"retain_envelope_fields":    types.BoolValue(true),
			"xml_root_element":          types.StringNull(),
		})
		result := s3LogStreamTypeOptions(obj)
		assert.NotNil(t, result)
		assert.Empty(t, result.JsonArrayEnvelopeField)
		assert.True(t, result.RetainEnvelopeFields)
		assert.Empty(t, result.XmlRootElement)
	})

	t.Run("all fields set", func(t *testing.T) {
		obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
			"json_array_envelope_field": types.StringValue("Records"),
			"retain_envelope_fields":    types.BoolValue(true),
			"xml_root_element":          types.StringValue("Events"),
		})
		result := s3LogStreamTypeOptions(obj)
		assert.NotNil(t, result)
		assert.Equal(t, "Records", result.JsonArrayEnvelopeField)
		assert.True(t, result.RetainEnvelopeFields)
		assert.Equal(t, "Events", result.XmlRootElement)
	})
}
