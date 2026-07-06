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

func TestRoleResource(t *testing.T) {
	roleName := "TerraformTestRole"
	roleUpdatedName := "TerraformTestRoleUpdated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccRoleResourceConfig(roleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_role.test", "name", roleName),
					resource.TestCheckResourceAttr("panther_role.test", "permissions.#", "1"),
					resource.TestCheckResourceAttrSet("panther_role.test", "id"),
					resource.TestCheckResourceAttrSet("panther_role.test", "created_at"),
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
				Config: providerConfig + testAccRoleResourceConfig(roleUpdatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_role.test", "name", roleUpdatedName),
				),
			},
		},
	})
}

func testAccRoleResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_role" "test" {
  name = %[1]q

  permissions = [
    "LogAnalysis:ReadData",
  ]
}
`, name)
}
