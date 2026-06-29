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

func TestUserResource(t *testing.T) {
	userEmail := fmt.Sprintf("test-%s@example.com", strings.ReplaceAll(uuid.NewString(), "-", ""))
	userUpdatedEmail := fmt.Sprintf("test-%s@example.com", strings.ReplaceAll(uuid.NewString(), "-", ""))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccUserResourceConfig(userEmail),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_user.test", "email", userEmail),
					resource.TestCheckResourceAttr("panther_user.test", "given_name", "Test"),
					resource.TestCheckResourceAttr("panther_user.test", "family_name", "User"),
					resource.TestCheckResourceAttr("panther_user.test", "role.name", "Analyst"),
					resource.TestCheckResourceAttrSet("panther_user.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccUserResourceConfig(userUpdatedEmail),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_user.test", "email", userUpdatedEmail),
				),
			},
		},
	})
}

func testAccUserResourceConfig(email string) string {
	return fmt.Sprintf(`
resource "panther_user" "test" {
  email       = %[1]q
  given_name  = "Test"
  family_name = "User"
  role = {
    name = "Analyst"
  }
}
`, email)
}