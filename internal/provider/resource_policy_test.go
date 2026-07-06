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

func TestPolicyResource(t *testing.T) {
	policyName := strings.ReplaceAll(uuid.NewString(), "-", "")
	policyUpdatedName := strings.ReplaceAll(uuid.NewString(), "-", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccPolicyResourceConfig(policyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_policy.test", "display_name", policyName),
					resource.TestCheckResourceAttr("panther_policy.test", "enabled", "true"),
					resource.TestCheckResourceAttr("panther_policy.test", "severity", "MEDIUM"),
					resource.TestCheckResourceAttr("panther_policy.test", "resource_types.#", "1"),
					resource.TestCheckResourceAttr("panther_policy.test", "resource_types.0", "AWS.S3.Bucket"),
					resource.TestCheckResourceAttrSet("panther_policy.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccPolicyResourceConfig(policyUpdatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_policy.test", "display_name", policyUpdatedName),
				),
			},
		},
	})
}

func testAccPolicyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_policy" "test" {
  display_name   = %[1]q
  body           = "def policy(resource): return True"
  enabled        = true
  resource_types = ["AWS.S3.Bucket"]
  severity       = "MEDIUM"
  tags           = ["test", "terraform"]
}
`, name)
}
