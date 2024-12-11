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
	"github.com/google/uuid"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestHttpSourceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testHttpSourceResourceConfig(strings.ReplaceAll(uuid.NewString(), "-", "")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_httpsource.test", "integration_label", ""),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_stream_type", "Auto"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_types", "[\"AWS.CloudFrontAccess\"]"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_type", "SharedSecret"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_header_key", "x-api-key"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_secret_value", "test-secret-value"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "panther_s3_source.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testHttpSourceResourceConfig("test-source-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_http_source.test", "name", "test-http-source"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testHttpSourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_httpsource" "test" {
  integration_label     = "%v"
  log_stream_type       = "Auto"
  log_types             = ["AWS.CloudFrontAccess"]
  security_type         = "SharedSecret"
  security_header_key   = "x-api-key"
  security_secret_value = "test-secret-value"
}
`, name)
}
