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
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"terraform-provider-panther/internal/client"
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

// TestS3SourceResource runs a comprehensive acceptance test covering the full lifecycle:
//
//	Step 1 — Create basic S3 source (TC1). Post-apply plan verifies no perpetual diffs (TC2).
//	Step 2 — Import by ID (TC10). Framework verifies plan after import shows no diff (TC11).
//	Step 3 — Update: change name, switch to CloudWatchLogs + retainEnvelopeFields, clear KMS key (TC3, TC4, TC7).
//	Step 4 — Update: multiple prefix_log_types, add KMS key, toggle managed notifications off (TC5, TC6, TC8).
//	Step 5 — Update: revert to Auto stream type, remove log_stream_type_options (TC9).
//	Step 6 — Drift detection: manually delete the resource out-of-band, verify Terraform proposes recreation (TC17).
//	Cleanup — TestCase automatically calls Delete, which succeeds on 404 (TC16).
func TestS3SourceResource(t *testing.T) {
	cfg, ok := loadS3TestConfig(t)
	if !ok {
		t.Skip("Skipping: PANTHER_S3_AWS_ACCOUNT_ID, PANTHER_S3_BUCKET_NAME, and PANTHER_S3_LOG_PROCESSING_ROLE_ARN must be set")
	}

	name := strings.ReplaceAll(uuid.NewString(), "-", "")
	nameUpdated := strings.ReplaceAll(uuid.NewString(), "-", "")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create basic S3 source (TC1).
			// The framework automatically runs a post-apply plan to verify no perpetual diffs (TC2).
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
			// Step 2: Import by ID (TC10). The framework verifies a subsequent plan shows no diff (TC11).
			{
				ResourceName:      "panther_s3_source.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update — change name, switch to CloudWatchLogs with retainEnvelopeFields,
			// clear KMS key (TC3, TC4, TC7).
			{
				Config: providerConfig + testS3SourceConfig_CloudWatchLogs(cfg, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_s3_source.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type", "CloudWatchLogs"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "log_stream_type_options.retain_envelope_fields", "true"),
					resource.TestCheckResourceAttr("panther_s3_source.test", "kms_key_arn", ""),
				),
			},
			// Step 4: Update — multiple prefix_log_types, add KMS key back,
			// toggle managed notifications off (TC5, TC6, TC8).
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
			// restore managed notifications (TC9).
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
			// Terraform's Read detects 404 and proposes recreation (TC17).
			// TestCase cleanup then calls Delete — succeeds because 404 is treated as
			// success by handleDeleteError (TC16).
			{
				Config:             providerConfig + testS3SourceConfig_RevertAuto(cfg, nameUpdated),
				Check:              manuallyDeleteS3Source(t),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// manuallyDeleteS3Source deletes the S3 source directly via the REST API (bypassing Terraform)
// to simulate out-of-band deletion for drift detection testing.
func manuallyDeleteS3Source(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["panther_s3_source.test"]
		if !ok {
			return fmt.Errorf("not found: panther_s3_source.test")
		}
		if rs.Primary.ID == "" {
			return errors.New("S3 source ID is not set")
		}
		url := os.Getenv("PANTHER_API_URL") + s3SourcePath + "/" + rs.Primary.ID
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			return fmt.Errorf("could not create delete request: %w", err)
		}
		req.Header.Set("X-API-Key", os.Getenv("PANTHER_API_TOKEN"))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("could not delete S3 source: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("expected 204 deleting S3 source, got %d", resp.StatusCode)
		}
		t.Logf("Manually deleted S3 source %s for drift detection test", rs.Primary.ID)
		return nil
	}
}

// --- Test configs ---

func testS3SourceConfig_Basic(cfg s3TestConfig, name string) string {
	return fmt.Sprintf(`
resource "panther_s3_source" "test" {
  aws_account_id                               = %q
  name                                         = %q
  log_processing_role_arn                       = %q
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
  log_processing_role_arn                       = %q
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
  log_processing_role_arn                       = %q
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
  log_processing_role_arn                       = %q
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
	attrTypes := map[string]attr.Type{
		"json_array_envelope_field": types.StringType,
		"retain_envelope_fields":    types.BoolType,
		"xml_root_element":          types.StringType,
	}

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
