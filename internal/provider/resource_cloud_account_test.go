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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCloudAccountResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccCloudAccountResourceConfig("999999999999", "Test Cloud Account", "arn:aws:iam::999999999999:role/PantherAuditRole", "panther-stack"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_cloud_account.test", "aws_account_id", "999999999999"),
					resource.TestCheckResourceAttr("panther_cloud_account.test", "label", "Test Cloud Account"),
					resource.TestCheckResourceAttr("panther_cloud_account.test", "audit_role", "arn:aws:iam::999999999999:role/PantherAuditRole"),
					resource.TestCheckResourceAttrSet("panther_cloud_account.test", "aws_stack_name"),
					resource.TestCheckResourceAttrSet("panther_cloud_account.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_cloud_account.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccCloudAccountResourceConfig("999999999999", "Updated Cloud Account", "arn:aws:iam::999999999999:role/PantherAuditRoleUpdated", "panther-stack-v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_cloud_account.test", "label", "Updated Cloud Account"),
					resource.TestCheckResourceAttr("panther_cloud_account.test", "audit_role", "arn:aws:iam::999999999999:role/PantherAuditRoleUpdated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCloudAccountResourceConfig(awsAccountID, label, auditRole, stackName string) string {
	return fmt.Sprintf(`
resource "panther_cloud_account" "test" {
  aws_account_id = "%s"
  label          = "%s"
  audit_role     = "%s"
}
`, awsAccountID, label, auditRole)
}