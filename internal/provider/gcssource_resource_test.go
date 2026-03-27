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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests for the panther_gcssource resource.
//
// These tests run against a LIVE Panther instance and perform real API calls:
//   - Create a GCS log source integration in Panther
//   - Read it back and verify all fields (including server-derived credentials_type)
//   - Import the resource by ID (credentials are lost — API returns empty string)
//   - Update the integration (change label, log stream type, add log_stream_type_options)
//   - Delete the integration (automatic cleanup by the test framework)
//
// Required env vars (in addition to PANTHER_API_URL and PANTHER_API_TOKEN):
//
//	Service Account tests:
//	  PANTHER_GCS_SA_CREDENTIALS_FILE  — path to a GCP service account JSON keyfile
//	  PANTHER_GCS_SA_PROJECT_ID        — GCP project ID containing the subscription
//	  PANTHER_GCS_SA_SUBSCRIPTION_ID   — Pub/Sub subscription for GCS bucket notifications
//	  PANTHER_GCS_SA_BUCKET            — GCS bucket name
//
//	WIF tests:
//	  PANTHER_GCS_WIF_CREDENTIALS_FILE — path to a GCP WIF credential config JSON
//	  PANTHER_GCS_WIF_PROJECT_ID       — GCP project ID containing the subscription
//	  PANTHER_GCS_WIF_SUBSCRIPTION_ID  — Pub/Sub subscription for GCS bucket notifications
//	  PANTHER_GCS_WIF_BUCKET           — GCS bucket name
//
// Tests are skipped when the corresponding env vars are not set.

// loadGcsTestConfig reads credentials from a file and returns the test configuration.
func loadGcsTestConfig(t *testing.T, credentialsFileEnv, projectIdEnv, subscriptionIdEnv, bucketEnv string) (credentials, projectId, subscriptionId, bucket string, ok bool) {
	t.Helper()
	credentialsFile := os.Getenv(credentialsFileEnv)
	projectId = os.Getenv(projectIdEnv)
	subscriptionId = os.Getenv(subscriptionIdEnv)
	bucket = os.Getenv(bucketEnv)

	if credentialsFile == "" || projectId == "" || subscriptionId == "" || bucket == "" {
		return "", "", "", "", false
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

	t.Logf("Loaded config:\n  project=%s\n  subscription=%s\n  bucket=%s\n  credentials=%s (%d bytes)",
		projectId, subscriptionId, bucket, credentialsFile, len(credentialsBytes))

	return string(credentialsBytes), projectId, subscriptionId, bucket, true
}

// TestGcsSourceResource_ServiceAccount groups tests that use the same GCS bucket + subscription.
// Subtests run sequentially because the Panther API enforces one integration per subscription.
func TestGcsSourceResource_ServiceAccount(t *testing.T) {
	t.Parallel()
	credentials, projectId, subscriptionId, bucket, ok := loadGcsTestConfig(t,
		"PANTHER_GCS_SA_CREDENTIALS_FILE",
		"PANTHER_GCS_SA_PROJECT_ID",
		"PANTHER_GCS_SA_SUBSCRIPTION_ID",
		"PANTHER_GCS_SA_BUCKET",
	)
	if !ok {
		t.Skip("Skipping: PANTHER_GCS_SA_CREDENTIALS_FILE, PANTHER_GCS_SA_PROJECT_ID, PANTHER_GCS_SA_SUBSCRIPTION_ID, and PANTHER_GCS_SA_BUCKET must be set")
	}

	t.Run("FullCRUD", func(t *testing.T) {
		runGcsSourceTest(t, credentials, projectId, subscriptionId, bucket, "service_account")
	})
}

// TestGcsSourceResource_WIF tests the full CRUD lifecycle using Workload Identity Federation.
func TestGcsSourceResource_WIF(t *testing.T) {
	t.Parallel()
	credentials, projectId, subscriptionId, bucket, ok := loadGcsTestConfig(t,
		"PANTHER_GCS_WIF_CREDENTIALS_FILE",
		"PANTHER_GCS_WIF_PROJECT_ID",
		"PANTHER_GCS_WIF_SUBSCRIPTION_ID",
		"PANTHER_GCS_WIF_BUCKET",
	)
	if !ok {
		t.Skip("Skipping: PANTHER_GCS_WIF_CREDENTIALS_FILE, PANTHER_GCS_WIF_PROJECT_ID, PANTHER_GCS_WIF_SUBSCRIPTION_ID, and PANTHER_GCS_WIF_BUCKET must be set")
	}

	runGcsSourceTest(t, credentials, projectId, subscriptionId, bucket, "wif")
}

func runGcsSourceTest(t *testing.T, credentials, projectId, subscriptionId, bucket, expectedCredentialsType string) {
	t.Helper()
	credType := expectedCredentialsType
	if credType == "service_account" {
		credType = "sa"
	}
	integrationLabel := fmt.Sprintf("tf-automated-test-gcs-%s", credType)
	integrationUpdatedLabel := fmt.Sprintf("tf-automated-test-gcs-%s-updated", credType)

	t.Logf("CRUD test (credentials_type=%s): create=%s, update=%s", expectedCredentialsType, integrationLabel, integrationUpdatedLabel)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config:    providerConfig + testGcsSourceResourceConfig(integrationLabel, subscriptionId, projectId, bucket, credentials, expectedCredentialsType),
				PreConfig: func() { t.Log("Step 1/4: Create (POST /log-sources/gcs)") },
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_gcssource.test", "integration_label", integrationLabel),
					resource.TestCheckResourceAttr("panther_gcssource.test", "subscription_id", subscriptionId),
					resource.TestCheckResourceAttr("panther_gcssource.test", "project_id", projectId),
					resource.TestCheckResourceAttr("panther_gcssource.test", "gcs_bucket", bucket),
					resource.TestCheckResourceAttr("panther_gcssource.test", "log_stream_type", "Auto"),
					resource.TestCheckResourceAttr("panther_gcssource.test", "credentials_type", expectedCredentialsType),
					resource.TestCheckResourceAttr("panther_gcssource.test", "prefix_log_types.0.prefix", ""),
					resource.TestCheckResourceAttr("panther_gcssource.test", "prefix_log_types.0.log_types.0", "GCP.AuditLog"),
					resource.TestCheckResourceAttrSet("panther_gcssource.test", "id"),
				),
			},
			// Step 2: Import
			{
				PreConfig:               func() { t.Log("Step 2/4: Import (GET /log-sources/gcs/{id})") },
				ResourceName:            "panther_gcssource.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credentials"},
			},
			// Step 3: Update — change label, log stream type, add log_stream_type_options, modify prefix
			{
				PreConfig: func() {
					t.Logf("Step 3/4: Update (PUT /log-sources/gcs/{id})\n  label=%s, log_stream_type=JsonArray, +log_stream_type_options", integrationUpdatedLabel)
				},
				Config: providerConfig + testGcsSourceUpdatedResourceConfig(integrationUpdatedLabel, subscriptionId, projectId, bucket, credentials, expectedCredentialsType),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("panther_gcssource.test", "integration_label", integrationUpdatedLabel),
					resource.TestCheckResourceAttr("panther_gcssource.test", "gcs_bucket", bucket),
					resource.TestCheckResourceAttr("panther_gcssource.test", "log_stream_type", "JsonArray"),
					resource.TestCheckResourceAttr("panther_gcssource.test", "log_stream_type_options.json_array_envelope_field", "records"),
					resource.TestCheckResourceAttr("panther_gcssource.test", "prefix_log_types.0.prefix", "logs/"),
					resource.TestCheckResourceAttr("panther_gcssource.test", "prefix_log_types.0.log_types.0", "GCP.AuditLog"),
					resource.TestCheckResourceAttr("panther_gcssource.test", "prefix_log_types.0.excluded_prefixes.0", "logs/tmp/*"),
					resource.TestCheckResourceAttr("panther_gcssource.test", "credentials_type", expectedCredentialsType),
				),
			},
			// Step 4: Delete (automatic)
		},
	})
	t.Log("Step 4/4: Delete (DELETE /log-sources/gcs/{id})")
}

func testGcsSourceResourceConfig(name, subscriptionId, projectId, bucket, credentials, credentialsType string) string {
	return fmt.Sprintf(`
resource "panther_gcssource" "test" {
  integration_label = "%s"
  subscription_id   = "%s"
  project_id        = "%s"
  gcs_bucket        = "%s"
  credentials       = %q
  credentials_type  = "%s"
  log_stream_type   = "Auto"

  prefix_log_types = [{
    prefix    = ""
    log_types = ["GCP.AuditLog"]
  }]
}
`, name, subscriptionId, projectId, bucket, credentials, credentialsType)
}

func testGcsSourceUpdatedResourceConfig(name, subscriptionId, projectId, bucket, credentials, credentialsType string) string {
	return fmt.Sprintf(`
resource "panther_gcssource" "test" {
  integration_label = "%s"
  subscription_id   = "%s"
  project_id        = "%s"
  gcs_bucket        = "%s"
  credentials       = %q
  credentials_type  = "%s"
  log_stream_type   = "JsonArray"

  log_stream_type_options = {
    json_array_envelope_field = "records"
  }

  prefix_log_types = [{
    prefix            = "logs/"
    log_types         = ["GCP.AuditLog"]
    excluded_prefixes = ["logs/tmp/*"]
  }]
}
`, name, subscriptionId, projectId, bucket, credentials, credentialsType)
}
