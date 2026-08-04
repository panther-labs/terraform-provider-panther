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
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestRoleResource(t *testing.T) {
	roleName := "tf-acc-test-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	roleUpdatedName := "tf-acc-test-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testRoleResourceConfig(roleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_role.test", "name", roleName),
					resource.TestCheckResourceAttr("panther_role.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("panther_role.test", "permissions.0", "AlertRead"),
					resource.TestCheckResourceAttr("panther_role.test", "permissions.1", "RuleRead"),
					resource.TestCheckResourceAttr("panther_role.test", "log_type_access_kind", "ALLOW_ALL"),
					resource.TestCheckResourceAttrSet("panther_role.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testUpdatedRoleResourceConfig(roleUpdatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_role.test", "name", roleUpdatedName),
					resource.TestCheckResourceAttr("panther_role.test", "permissions.#", "3"),
					resource.TestCheckResourceAttr("panther_role.test", "permissions.2", "SummaryRead"),
				),
			},
			// Drift detection: manually delete the role, then verify Read detects 404
			// and removes it from state, causing a non-empty refresh plan (recreate).
			{
				Config:             providerConfig + testUpdatedRoleResourceConfig(roleUpdatedName),
				Check:              manuallyDeleteSource(t, "panther_role.test", rolePath),
				ExpectNonEmptyPlan: true,
			},
			// TestCase cleanup calls Delete automatically — succeeds because 404 is treated as success.
		},
	})
}

func testRoleResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_role" "test" {
  name = "%v"
  permissions = [
    "AlertRead",
    "RuleRead",
  ]
  log_type_access_kind = "ALLOW_ALL"
}
`, name)
}

func testUpdatedRoleResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_role" "test" {
  name = "%v"
  permissions = [
    "AlertRead",
    "RuleRead",
    "SummaryRead",
  ]
  log_type_access_kind = "ALLOW_ALL"
}
`, name)
}
