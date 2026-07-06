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

func TestSimpleRuleResource(t *testing.T) {
	simpleRuleName := strings.ReplaceAll(uuid.NewString(), "-", "")
	simpleRuleUpdatedName := strings.ReplaceAll(uuid.NewString(), "-", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSimpleRuleResourceConfig(simpleRuleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_simple_rule.test", "display_name", simpleRuleName),
					resource.TestCheckResourceAttr("panther_simple_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttr("panther_simple_rule.test", "severity", "CRITICAL"),
					resource.TestCheckResourceAttr("panther_simple_rule.test", "log_types.#", "1"),
					resource.TestCheckResourceAttr("panther_simple_rule.test", "log_types.0", "AWS.CloudTrail"),
					resource.TestCheckResourceAttr("panther_simple_rule.test", "dedup_period_minutes", "60"),
					resource.TestCheckResourceAttr("panther_simple_rule.test", "threshold", "1"),
					resource.TestCheckResourceAttrSet("panther_simple_rule.test", "id"),
					resource.TestCheckResourceAttrSet("panther_simple_rule.test", "detection"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_simple_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccSimpleRuleResourceConfig(simpleRuleUpdatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_simple_rule.test", "display_name", simpleRuleUpdatedName),
				),
			},
		},
	})
}

func testAccSimpleRuleResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_simple_rule" "test" {
  display_name         = %[1]q
  detection            = <<-EOT
    MatchFilters:
      - Key: eventName
        Condition: Equals
        Values:
          - ConsoleLogin
  EOT
  enabled              = true
  log_types            = ["AWS.CloudTrail"]
  severity             = "CRITICAL"
  dedup_period_minutes = 60
  threshold            = 1
  tags                 = ["test", "terraform"]
}
`, name)
}
