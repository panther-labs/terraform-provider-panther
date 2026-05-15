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
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-panther/internal/client"
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
					resource.TestCheckResourceAttr("panther_httpsource.test", "auth_method", "SharedSecret"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "auth_header_key", "x-api-key"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "auth_secret_value", "test-secret-value"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "panther_httpsource.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret_value", "auth_password", "auth_bearer_token"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testUpdatedHttpSourceResourceConfig(integrationUpdatedLabel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_httpsource.test", "integration_label", integrationUpdatedLabel),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_stream_type", "JSON"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_types.0", "Zscaler.ZIA.WebLog"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "auth_method", "Basic"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "auth_username", "foo"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "auth_password", "bar"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_stream_type_options.json_array_envelope_field", "records"),
					resource.TestCheckResourceAttr("panther_httpsource.test", "log_stream_type_options.xml_root_element", "root"),
				),
			},
			// Drift detection: manually delete the resource, then verify Read detects 404
			// and removes it from state, causing a non-empty refresh plan (recreate).
			{
				Config:             providerConfig + testUpdatedHttpSourceResourceConfig(integrationUpdatedLabel),
				Check:              manuallyDeleteSource(t, "panther_httpsource.test", httpSourcePath),
				ExpectNonEmptyPlan: true,
			},
			// TestCase cleanup calls Delete automatically — succeeds because 404 is treated as success.
		},
	})
}

func testHttpSourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_httpsource" "test" {
  integration_label     = "%v"
  log_stream_type       = "Auto"
  log_types             = ["AWS.CloudFrontAccess"]
  auth_method           = "SharedSecret"
  auth_header_key       = "x-api-key"
  auth_secret_value     = "test-secret-value"
}
`, name)
}

func testUpdatedHttpSourceResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "panther_httpsource" "test" {
  integration_label     = "%v"
  log_stream_type       = "JSON"
  log_types             = ["Zscaler.ZIA.WebLog"]
  auth_method         = "Basic"
  auth_username   	= "foo"
  auth_password 	= "bar"
  log_stream_type_options = {
    json_array_envelope_field = "records"
	xml_root_element = "root"
  }
}
`, name)
}

// manuallyDeleteSource issues DELETEs against the given source's REST path with a
// bounded retry loop, treating 5xx as retryable. The retry exists because the
// underlying Kinesis Firehose can be in a CREATING state immediately after Create,
// during which DELETE returns 5xx until the stream transitions to ACTIVE. 404 is
// treated as success — the resource is already gone. resourceName is the
// Terraform-state address (e.g. "panther_httpsource.parent"); basePath is the REST
// collection path (e.g. httpSourcePath).
func manuallyDeleteSource(t *testing.T, resourceName, basePath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("%s ID is not set", resourceName)
		}
		c := client.NewRESTClient(os.Getenv("PANTHER_API_URL"), os.Getenv("PANTHER_API_TOKEN"))
		path := basePath + "/" + rs.Primary.ID
		const maxRetries = 10
		for retry := 0; retry < maxRetries; retry++ {
			err := client.RestDelete(context.Background(), c, path)
			if err == nil || client.IsNotFound(err) {
				return nil
			}
			var apiErr *client.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode < 500 {
				return fmt.Errorf("could not delete %s: %w", resourceName, err)
			}
			t.Logf("Could not delete %s %s with retry %d: %v. retrying\n", resourceName, rs.Primary.ID, retry, err)
			time.Sleep(5 * time.Second)
		}
		return fmt.Errorf("could not delete %s after %d retries", resourceName, maxRetries)
	}
}
