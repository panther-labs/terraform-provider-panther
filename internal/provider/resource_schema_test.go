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
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSchemaResource(t *testing.T) {
	// Use shorter names as the API has length limits for schema names
	schemaName := fmt.Sprintf("Custom.Test%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSchemaResourceConfig(schemaName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_schema.test", "name", schemaName),
					resource.TestCheckResourceAttr("panther_schema.test", "description", "Test schema for Terraform provider"),
					resource.TestCheckResourceAttr("panther_schema.test", "is_field_discovery_enabled", "true"),
					resource.TestCheckResourceAttrSet("panther_schema.test", "id"),
					resource.TestCheckResourceAttrSet("panther_schema.test", "version"),
					resource.TestCheckResourceAttrSet("panther_schema.test", "revision"),
					resource.TestCheckResourceAttrSet("panther_schema.test", "spec"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_schema.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore spec field during import verification due to API normalization
				ImportStateVerifyIgnore: []string{"spec"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccSchemaResourceConfigUpdated(schemaName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_schema.test", "name", schemaName),
					resource.TestCheckResourceAttr("panther_schema.test", "description", "Updated test schema for Terraform provider"),
					resource.TestCheckResourceAttr("panther_schema.test", "is_field_discovery_enabled", "false"),
					resource.TestCheckResourceAttrSet("panther_schema.test", "id"),
					resource.TestCheckResourceAttrSet("panther_schema.test", "version"),
					resource.TestCheckResourceAttrSet("panther_schema.test", "revision"),
				),
			},
		},
	})
}

func testAccSchemaResourceConfig(schemaName string) string {
	return fmt.Sprintf(`
resource "panther_schema" "test" {
  name        = %[1]q
  description = "Test schema for Terraform provider"
  spec = <<EOF
schema: %[1]s
fields:
  - name: timestamp
    type: timestamp
    timeFormats:
      - unix
    isEventTime: true
  - name: message
    type: string
  - name: level
    type: string
EOF
  is_field_discovery_enabled = true
}
`, schemaName)
}

func testAccSchemaResourceConfigUpdated(schemaName string) string {
	return fmt.Sprintf(`
resource "panther_schema" "test" {
  name        = %[1]q
  description = "Updated test schema for Terraform provider"
  spec = <<EOF
schema: %[1]s
fields:
  - name: timestamp
    type: timestamp
    timeFormats:
      - unix
    isEventTime: true
  - name: message
    type: string
  - name: level
    type: string
  - name: user_id
    type: string
EOF
  is_field_discovery_enabled = false
}
`, schemaName)
}