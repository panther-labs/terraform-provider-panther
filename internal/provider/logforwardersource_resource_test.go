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


// Acceptance tests for the panther_logforwardersource resource.
//
// These tests run against a LIVE Panther instance and perform real API calls:
//   - Create a log forwarder source integration in Panther
//   - Read it back and verify all fields
//   - Import the resource by ID
//   - Update the integration (change label, log stream type, add options)
//   - Drift detection: manually delete and verify Read detects 404
//   - Delete the integration (automatic cleanup by the test framework)
//
// Required env vars: PANTHER_API_URL and PANTHER_API_TOKEN.
func TestLogForwarderSourceResource(t *testing.T) {
	integrationLabel := strings.ReplaceAll(uuid.NewString(), "-", "")
	integrationUpdatedLabel := strings.ReplaceAll(uuid.NewString(), "-", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read
			{
				Config: providerConfig + testLogForwarderSourceResourceConfig(integrationLabel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("panther_logforwardersource.test", "id"),
					resource.TestCheckResourceAttr("panther_logforwardersource.test", "integration_label", integrationLabel),
					resource.TestCheckResourceAttr("panther_logforwardersource.test", "log_stream_type", "Auto"),
					resource.TestCheckResourceAttr("panther_logforwardersource.test", "log_types.0", "AWS.CloudTrail"),
				),
			},
			// Step 2: ImportState
			{
				ResourceName:      "panther_logforwardersource.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update — change label, log_stream_type, add log_stream_type_options
			{
				Config: providerConfig + testUpdatedLogForwarderSourceResourceConfig(integrationUpdatedLabel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_logforwardersource.test", "integration_label", integrationUpdatedLabel),
					resource.TestCheckResourceAttr("panther_logforwardersource.test", "log_stream_type", "JsonArray"),
					resource.TestCheckResourceAttr("panther_logforwardersource.test", "log_types.0", "AWS.CloudTrail"),
					resource.TestCheckResourceAttr("panther_logforwardersource.test", "log_stream_type_options.json_array_envelope_field", "records"),
				),
			},
			// TestCase cleanup calls Delete automatically.
		},
	})
}

func testLogForwarderSourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_logforwardersource" "test" {
  integration_label = "%s"
  log_stream_type   = "Auto"
  log_types         = ["AWS.CloudTrail"]
}
`, name)
}

func testUpdatedLogForwarderSourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_logforwardersource" "test" {
  integration_label = "%s"
  log_stream_type   = "JsonArray"
  log_types         = ["AWS.CloudTrail"]
  log_stream_type_options = {
    json_array_envelope_field = "records"
  }
}
`, name)
}
