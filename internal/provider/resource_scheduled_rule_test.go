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

func TestScheduledRuleResource(t *testing.T) {
	scheduledRuleName := strings.ReplaceAll(uuid.NewString(), "-", "")
	scheduledRuleUpdatedName := strings.ReplaceAll(uuid.NewString(), "-", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccScheduledRuleResourceConfig(scheduledRuleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "display_name", scheduledRuleName),
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "severity", "HIGH"),
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "scheduled_queries.#", "1"),
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "scheduled_queries.0", "test-query"),
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "dedup_period_minutes", "60"),
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "threshold", "1"),
					resource.TestCheckResourceAttrSet("panther_scheduled_rule.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_scheduled_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccScheduledRuleResourceConfig(scheduledRuleUpdatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_scheduled_rule.test", "display_name", scheduledRuleUpdatedName),
				),
			},
		},
	})
}

func testAccScheduledRuleResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_scheduled_rule" "test" {
  display_name         = %[1]q
  body                 = "def rule(event): return True"
  enabled              = true
  scheduled_queries    = ["test-query"]
  severity             = "HIGH"
  dedup_period_minutes = 60
  threshold            = 1
  tags                 = ["test", "terraform"]
}
`, name)
}
