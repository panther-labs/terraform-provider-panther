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
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"net/http"
	"os"
	"regexp"
	"strings"
	"terraform-provider-panther/internal/client/panther"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// There is a caveat in testing the http source resource with an acceptance test, because after running all the steps the test case will
// try to delete the resource. For a http resource that's not possible directly after creating it because the underlying firehose stream
// is in a CREATING state and cannot be deleted. This is why the test case has an extra step which will retry deleting it manually until
// it succeeds. We then need to catch the error from the post apply refresh plan step and ignore it, as that step cannot be skipped.
func TestHttpSourceResource(t *testing.T) {
	integrationLabel := strings.ReplaceAll(uuid.NewString(), "-", "")
	integrationUpdatedLabel := strings.ReplaceAll(uuid.NewString(), "-", "")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testHttpSourceResourceConfig(integrationLabel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_httpsource.test", "integration_label", integrationLabel),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_stream_type", "Auto"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_types.0", "AWS.CloudFrontAccess"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_type", "SharedSecret"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_header_key", "x-api-key"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_secret_value", "test-secret-value"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "panther_httpsource.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"security_secret_value", "security_password"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testUpdatedHttpSourceResourceConfig(integrationUpdatedLabel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_httpsource.test", "integration_label", integrationUpdatedLabel),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_stream_type", "JSON"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_types.0", "Zscaler.ZIA.WebLog"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_type", "Basic"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_username", "foo"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "security_password", "bar"),
				),
			},
			// Provide an unchanged configuration and manually delete the resource
			{
				Config:      providerConfig + testHttpSourceResourceConfig(integrationUpdatedLabel),
				Check:       manuallyDeleteSource,
				ExpectError: regexp.MustCompile("Error running post-apply refresh plan"),
			},
			// Delete testing automatically occurs in TestCase, in our case it is already deleted and the delete step
			// succeeds as the method is idempotent
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

func testUpdatedHttpSourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_httpsource" "test" {
  integration_label     = "%v"
  log_stream_type       = "JSON"
  log_types             = ["Zscaler.ZIA.WebLog"]
  security_type         = "Basic"
  security_username   	= "foo"
  security_password 	= "bar"
}
`, name)
}

func manuallyDeleteSource(s *terraform.State) error {
	httpSource, ok := s.RootModule().Resources["panther_httpsource.test"]
	if !ok {
		return fmt.Errorf("not found: %s", "panther_httpsource.test")
	}
	if httpSource.Primary.ID == "" {
		return errors.New("http source ID is not set")
	}
	url := os.Getenv("PANTHER_API_URL") + panther.RestHttpSourcePath + "/" + httpSource.Primary.ID
	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("could not create delete request: %w", err)
	}
	req.Header.Set("X-API-Key", os.Getenv("PANTHER_API_TOKEN"))
	retry := 0
	for retry < 10 {
		response, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("could not delete http source: %w", err)
		}
		if response.StatusCode == http.StatusNoContent {
			return nil
		}
		time.Sleep(5 * time.Second)
		retry++
	}

	return fmt.Errorf("could not delete http source after %d retries", retry)
}
