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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests for the panther_pubsubsource resource.
//
// These tests run against a LIVE Panther instance and perform real API calls:
//   - Create a Pub/Sub log source integration in Panther
//   - Read it back and verify all fields (including server-derived credentials_type)
//   - Import the resource by ID (credentials are lost — API returns empty string)
//   - Update the integration (change label, log stream type, add options)
//   - Delete the integration (automatic cleanup by the test framework)
//
// The Panther API validates GCP credentials via a health check on create/update
// (parses credentials JSON, connects to GCP, verifies the subscription exists,
// and checks IAM permissions). Dummy credentials will be rejected.
//
// Two test functions cover both GCP credential types:
//   - TestPubSubSourceResource_ServiceAccount: creates a source with a GCP service account key
//     → expects credentials_type = "service_account"
//   - TestPubSubSourceResource_WIF: creates a source with Workload Identity Federation config
//     → expects credentials_type = "wif"
//
// Required env vars: see .env.test and .env.pubsub.test

// loadPubSubTestConfig reads credentials from a file and returns the test configuration.
// Returns empty strings if any required env var is missing.
// Relative file paths are resolved from the repo root (go test sets CWD to the package directory).
func loadPubSubTestConfig(t *testing.T, credentialsFileEnv, projectIdEnv, subscriptionIdEnv string) (credentials, projectId, subscriptionId string, ok bool) {
	t.Helper()
	credentialsFile := os.Getenv(credentialsFileEnv)
	projectId = os.Getenv(projectIdEnv)
	subscriptionId = os.Getenv(subscriptionIdEnv)

	if credentialsFile == "" || projectId == "" || subscriptionId == "" {
		return "", "", "", false
	}

	// Resolve relative paths from repo root (where go.mod lives)
	if !filepath.IsAbs(credentialsFile) {
		if root := findRepoRoot(); root != "" {
			credentialsFile = filepath.Join(root, credentialsFile)
		}
	}

	credentialsBytes, err := os.ReadFile(credentialsFile)
	if err != nil {
		t.Fatalf("Env vars are set but credentials file is unreadable: %s: %v", credentialsFile, err)
	}

	t.Logf("Loaded config:\n  project=%s\n  subscription=%s\n  credentials=%s (%d bytes)", projectId, subscriptionId, credentialsFile, len(credentialsBytes))

	return string(credentialsBytes), projectId, subscriptionId, true
}

// repoRoot is cached to avoid redundant filesystem walks when multiple tests call loadPubSubTestConfig.
var (
	repoRoot     string
	repoRootOnce sync.Once
)

// findRepoRoot walks up from the current directory looking for go.mod (cached after first call).
func findRepoRoot() string {
	repoRootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			return
		}
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				repoRoot = dir
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	})
	return repoRoot
}

// TestPubSubSourceResource_ServiceAccount tests the full CRUD lifecycle using a GCP service account key.
// Creates a real Pub/Sub source in Panther, verifies credentials_type = "service_account",
// then updates and deletes it.
func TestPubSubSourceResource_ServiceAccount(t *testing.T) {
	t.Parallel()
	credentials, projectId, subscriptionId, ok := loadPubSubTestConfig(t,
		"PANTHER_PUBSUB_SA_CREDENTIALS_FILE",
		"PANTHER_PUBSUB_SA_PROJECT_ID",
		"PANTHER_PUBSUB_SA_SUBSCRIPTION_ID",
	)
	if !ok {
		t.Skip("Skipping: PANTHER_PUBSUB_SA_CREDENTIALS_FILE, PANTHER_PUBSUB_SA_PROJECT_ID, and PANTHER_PUBSUB_SA_SUBSCRIPTION_ID must be set")
	}

	runPubSubSourceTest(t, credentials, projectId, subscriptionId, "service_account")
}

// TestPubSubSourceResource_SA_DerivedProjectId tests that project_id can be omitted for service
// account credentials — the API derives it from the keyfile's project_id field.
func TestPubSubSourceResource_SA_DerivedProjectId(t *testing.T) {
	t.Parallel()
	credentials, _, subscriptionId, ok := loadPubSubTestConfig(t,
		"PANTHER_PUBSUB_SA_CREDENTIALS_FILE",
		"PANTHER_PUBSUB_SA_PROJECT_ID",
		"PANTHER_PUBSUB_SA_SUBSCRIPTION_ID",
	)
	if !ok {
		t.Skip("Skipping: PANTHER_PUBSUB_SA_* env vars must be set")
	}

	integrationLabel := "tf-test-pubsub-sa-no-project"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "panther_pubsubsource" "test" {
  integration_label = "%s"
  subscription_id   = "%s"
  credentials       = %q
  credentials_type  = "service_account"
  log_types         = ["GCP.AuditLog"]
  log_stream_type   = "Auto"
}
`, integrationLabel, subscriptionId, credentials),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "integration_label", integrationLabel),
					// project_id was not provided — API should derive it from the SA keyfile
					resource.TestCheckResourceAttrSet("panther_pubsubsource.test", "project_id"),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "credentials_type", "service_account"),
				),
			},
		},
	})
}

// TestPubSubSourceResource_WIF tests the full CRUD lifecycle using Workload Identity Federation.
// Creates a real Pub/Sub source in Panther, verifies credentials_type = "wif",
// then updates and deletes it.
func TestPubSubSourceResource_WIF(t *testing.T) {
	t.Parallel()
	credentials, projectId, subscriptionId, ok := loadPubSubTestConfig(t,
		"PANTHER_PUBSUB_WIF_CREDENTIALS_FILE",
		"PANTHER_PUBSUB_WIF_PROJECT_ID",
		"PANTHER_PUBSUB_WIF_SUBSCRIPTION_ID",
	)
	if !ok {
		t.Skip("Skipping: PANTHER_PUBSUB_WIF_CREDENTIALS_FILE, PANTHER_PUBSUB_WIF_PROJECT_ID, and PANTHER_PUBSUB_WIF_SUBSCRIPTION_ID must be set")
	}

	runPubSubSourceTest(t, credentials, projectId, subscriptionId, "wif")
}

func runPubSubSourceTest(t *testing.T, credentials, projectId, subscriptionId, expectedCredentialsType string) {
	t.Helper()
	credType := expectedCredentialsType
	if credType == "service_account" {
		credType = "sa"
	}
	integrationLabel := fmt.Sprintf("tf-automated-test-pubsub-%s", credType)
	integrationUpdatedLabel := fmt.Sprintf("tf-automated-test-pubsub-%s-updated", credType)

	t.Logf("CRUD test (credentials_type=%s): create=%s, update=%s", expectedCredentialsType, integrationLabel, integrationUpdatedLabel)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create — POST /log-sources/pubsub
			// Creates a real Pub/Sub source integration in Panther. The API validates the
			// credentials against GCP (health check), then returns the integration ID and
			// derived credentials_type. Verifies all fields match the config.
			{
				Config:    providerConfig + testPubSubSourceResourceConfig(integrationLabel, subscriptionId, projectId, credentials, expectedCredentialsType),
				PreConfig: func() { t.Log("Step 1/4: Create (POST /log-sources/pubsub)") },
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "integration_label", integrationLabel),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "subscription_id", subscriptionId),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "project_id", projectId),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "log_stream_type", "Auto"),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "log_types.0", "GCP.AuditLog"),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "credentials_type", expectedCredentialsType),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "regional_endpoint", ""),
					resource.TestCheckResourceAttrSet("panther_pubsubsource.test", "id"),
				),
			},
			// Step 2: Import — GET /log-sources/pubsub/{id}
			// Imports the resource by ID and verifies all fields match state, EXCEPT credentials
			// (the API returns "" for this sensitive field, so it can't round-trip through import).
			{
				PreConfig:               func() { t.Log("Step 2/4: Import (GET /log-sources/pubsub/{id})") },
				ResourceName:            "panther_pubsubsource.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credentials"},
			},
			// Step 3: Update — PUT /log-sources/pubsub/{id}
			// Changes integration_label and log_stream_type (Auto → JSON), adds
			// log_stream_type_options. Verifies the API accepts the update and all
			// fields reflect the new values.
			{
				PreConfig: func() {
					t.Logf("Step 3/4: Update (PUT /log-sources/pubsub/{id})\n  label=%s, log_stream_type=JSON, +log_stream_type_options", integrationUpdatedLabel)
				},
				Config: providerConfig + testUpdatedPubSubSourceResourceConfig(integrationUpdatedLabel, subscriptionId, projectId, credentials, expectedCredentialsType),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "integration_label", integrationUpdatedLabel),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "subscription_id", subscriptionId),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "project_id", projectId),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "log_stream_type", "JSON"),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "log_types.0", "GCP.AuditLog"),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "log_stream_type_options.json_array_envelope_field", "records"),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "log_stream_type_options.xml_root_element", "root"),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "credentials_type", expectedCredentialsType),
					resource.TestCheckResourceAttr("panther_pubsubsource.test", "regional_endpoint", ""),
				),
			},
			// Step 4: Delete — DELETE /log-sources/pubsub/{id}
			// Automatic cleanup by the test framework. Pub/Sub deletion is immediate
			// (no Firehose delay like httpsource), so no manual retry is needed.
		},
	})
	// resource.Test deletes the resource before returning — log for completeness
	t.Log("Step 4/4: Delete (DELETE /log-sources/pubsub/{id})")
}

func testPubSubSourceResourceConfig(name, subscriptionId, projectId, credentials, credentialsType string) string {
	return fmt.Sprintf(`
resource "panther_pubsubsource" "test" {
  integration_label = "%s"
  subscription_id   = "%s"
  project_id        = "%s"
  credentials       = %q
  credentials_type  = "%s"
  log_types         = ["GCP.AuditLog"]
  log_stream_type   = "Auto"
}
`, name, subscriptionId, projectId, credentials, credentialsType)
}

func testUpdatedPubSubSourceResourceConfig(name, subscriptionId, projectId, credentials, credentialsType string) string {
	return fmt.Sprintf(`
resource "panther_pubsubsource" "test" {
  integration_label = "%s"
  subscription_id   = "%s"
  project_id        = "%s"
  credentials       = %q
  credentials_type  = "%s"
  log_types         = ["GCP.AuditLog"]
  log_stream_type   = "JSON"
  log_stream_type_options = {
    json_array_envelope_field = "records"
    xml_root_element          = "root"
  }
}
`, name, subscriptionId, projectId, credentials, credentialsType)
}
