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
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"terraform-provider-panther/internal/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// freshAwsAccountId returns a random 12-digit AWS account ID for test runs to
// avoid the 409-on-duplicate trap if two test runs overlap or state is left
// orphaned from a previous run.
func freshAwsAccountId() string {
	// #nosec G404 — non-cryptographic, test-only.
	return fmt.Sprintf("%012d", rand.Int63n(1_000_000_000_000))
}

func TestAwsCloudAccountResource(t *testing.T) {
	label := strings.ReplaceAll(uuid.NewString(), "-", "")[:32]
	updatedLabel := strings.ReplaceAll(uuid.NewString(), "-", "")[:32]
	accountID := freshAwsAccountId()
	auditRole := fmt.Sprintf("arn:aws:iam::%s:role/PantherAuditRole-tf-acc", accountID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkAwsCloudAccountDestroyed,
		Steps: []resource.TestStep{
			// Create + Read
			{
				Config: providerConfig + testAwsCloudAccountConfig(label, accountID, auditRole, nil, nil, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "integration_label", label),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "aws_account_id", accountID),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "aws_scan_config.audit_role", auditRole),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "region_ignore_list.#", "0"),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "resource_type_ignore_list.#", "0"),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "resource_regex_ignore_list.#", "0"),
				),
			},
			// ImportState — no sensitive fields, so no ImportStateVerifyIgnore needed.
			{
				ResourceName:      "panther_aws_cloud_account.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update integration_label + populate exclusion lists
			{
				Config: providerConfig + testAwsCloudAccountConfig(updatedLabel, accountID, auditRole,
					[]string{"us-east-1", "eu-west-2"},
					[]string{"AWS.S3.Bucket"},
					[]string{`^arn:aws:s3:::test-.*$`},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "integration_label", updatedLabel),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "region_ignore_list.#", "2"),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "region_ignore_list.0", "us-east-1"),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "region_ignore_list.1", "eu-west-2"),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "resource_type_ignore_list.0", "AWS.S3.Bucket"),
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "resource_regex_ignore_list.0", `^arn:aws:s3:::test-.*$`),
				),
			},
			// Drift detection on Read: out-of-band DELETE then expect non-empty plan.
			// (Resource's own Delete CRUD is asserted by CheckDestroy at end-of-test.)
			{
				Config:             providerConfig + testAwsCloudAccountConfig(updatedLabel, accountID, auditRole, nil, nil, nil),
				Check:              manuallyDeleteSource(t, "panther_aws_cloud_account.test", awsCloudAccountPath),
				ExpectNonEmptyPlan: true,
			},
			// Re-apply to recreate the resource drifted away above so the framework's
			// cleanup tears down a live resource (asserted via CheckDestroy).
			{
				Config: providerConfig + testAwsCloudAccountConfig(updatedLabel, accountID, auditRole, nil, nil, nil),
				Check:  resource.TestCheckResourceAttrSet("panther_aws_cloud_account.test", "id"),
			},
		},
	})
}

func TestAwsCloudAccountResource_ForceNew(t *testing.T) {
	label := strings.ReplaceAll(uuid.NewString(), "-", "")[:32]
	originalAccount := freshAwsAccountId()
	newAccount := freshAwsAccountId()
	originalRole := fmt.Sprintf("arn:aws:iam::%s:role/PantherAuditRole-tf-acc", originalAccount)
	newRole := fmt.Sprintf("arn:aws:iam::%s:role/PantherAuditRole-tf-acc", newAccount)

	var originalID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAwsCloudAccountConfig(label, originalAccount, originalRole, nil, nil, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "aws_account_id", originalAccount),
					resource.TestCheckResourceAttrWith("panther_aws_cloud_account.test", "id", func(v string) error {
						originalID = v
						if v == "" {
							return fmt.Errorf("original id is empty")
						}
						return nil
					}),
				),
			},
			{
				Config: providerConfig + testAwsCloudAccountConfig(label, newAccount, newRole, nil, nil, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_aws_cloud_account.test", "aws_account_id", newAccount),
					resource.TestCheckResourceAttrWith("panther_aws_cloud_account.test", "id", func(v string) error {
						if v == originalID {
							return fmt.Errorf("id did not change after aws_account_id replacement: %s", v)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAwsCloudAccountResource_PlanTimeValidation(t *testing.T) {
	const (
		validAccount = "123456789012"
		validRole    = "arn:aws:iam::123456789012:role/PantherAuditRole"
	)
	cfg := func(label, account, audit string, regions, regexes []string) string {
		return providerConfig + testAwsCloudAccountConfig(label, account, audit, regions, nil, regexes)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg("valid-label", "abc", validRole, nil, nil),
				ExpectError: regexp.MustCompile(`aws_account_id[\s\S]*\[0-9\]\{12\}`),
				PlanOnly:    true,
			},
			{
				Config:      cfg("bad/label", validAccount, validRole, nil, nil),
				ExpectError: regexp.MustCompile(`integration_label[\s\S]*0-9a-zA-Z`),
				PlanOnly:    true,
			},
			{
				Config:      cfg(strings.Repeat("a", 40), validAccount, validRole, nil, nil),
				ExpectError: regexp.MustCompile(`integration_label[\s\S]*at most 36`),
				PlanOnly:    true,
			},
			{
				Config:      cfg("valid-label", validAccount, "not-an-arn", nil, nil),
				ExpectError: regexp.MustCompile(`audit_role[\s\S]*valid IAM role ARN`),
				PlanOnly:    true,
			},
			{
				Config:      cfg("valid-label", validAccount, validRole, []string{"us-fake-1!"}, nil),
				ExpectError: regexp.MustCompile(`region_ignore_list[\s\S]*valid AWS region code`),
				PlanOnly:    true,
			},
			{
				Config:      cfg("valid-label", validAccount, validRole, nil, []string{"^[unclosed"}),
				ExpectError: regexp.MustCompile(`Invalid regular expression[\s\S]*resource_regex_ignore_list`),
				PlanOnly:    true,
			},
		},
	})
}

func testAwsCloudAccountConfig(label, account, auditRole string, regions, resourceTypes, resourceRegexes []string) string {
	return fmt.Sprintf(`
resource "panther_aws_cloud_account" "test" {
  integration_label = %q
  aws_account_id    = %q

  aws_scan_config = {
    audit_role = %q
  }

  region_ignore_list         = %s
  resource_type_ignore_list  = %s
  resource_regex_ignore_list = %s
}
`, label, account, auditRole, hclList(regions), hclList(resourceTypes), hclList(resourceRegexes))
}

func hclList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// checkAwsCloudAccountDestroyed verifies that each panther_aws_cloud_account tracked
// in the final state is gone server-side after the framework's auto-destroy step,
// mirroring checkS3SourceDestroyed in s3source_resource_test.go.
func checkAwsCloudAccountDestroyed(s *terraform.State) error {
	c := client.NewRESTClient(os.Getenv("PANTHER_API_URL"), os.Getenv("PANTHER_API_TOKEN"), testUserAgent)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "panther_aws_cloud_account" {
			continue
		}
		_, err := client.RestDo[client.AwsCloudAccount](context.Background(), c, http.MethodGet, awsCloudAccountPath+"/"+rs.Primary.ID, nil)
		if err == nil {
			return fmt.Errorf("AWS Cloud Account %s still exists after destroy", rs.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking AWS Cloud Account %s: %w", rs.Primary.ID, err)
		}
	}
	return nil
}
