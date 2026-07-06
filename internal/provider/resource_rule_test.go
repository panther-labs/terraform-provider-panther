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

func TestRuleResource(t *testing.T) {
	ruleName := strings.ReplaceAll(uuid.NewString(), "-", "")
	ruleUpdatedName := strings.ReplaceAll(uuid.NewString(), "-", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccRuleResourceConfig(ruleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_rule.test", "display_name", ruleName),
					resource.TestCheckResourceAttr("panther_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttr("panther_rule.test", "severity", "HIGH"),
					resource.TestCheckResourceAttr("panther_rule.test", "log_types.#", "1"),
					resource.TestCheckResourceAttr("panther_rule.test", "log_types.0", "AWS.VPCFlow"),
					resource.TestCheckResourceAttrSet("panther_rule.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore tags as the API may return them in a different order
				ImportStateVerifyIgnore: []string{"tags"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccRuleResourceConfig(ruleUpdatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_rule.test", "display_name", ruleUpdatedName),
				),
			},
		},
	})
}

func testAccRuleResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_rule" "test" {
  display_name = %[1]q
  body         = "def rule(event): return True"
  enabled      = true
  log_types    = ["AWS.VPCFlow"]
  severity     = "HIGH"
  tags         = ["test", "terraform"]
}
`, name)
}