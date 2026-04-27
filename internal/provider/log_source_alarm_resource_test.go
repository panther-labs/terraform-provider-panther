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
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-panther/internal/client"
)

// Acceptance tests for panther_log_source_alarm.
//
// Architecture: the single live lifecycle test (TestLogSourceAlarmResource) bundles
// every scenario that needs a parent log source — CRUD, import (valid and malformed),
// boundary thresholds, drift detection — into ONE resource.TestCase so the shared
// parent gcssource is created once and torn down once. Plan-time tests and the
// no-parent 404 test stay as separate top-level tests (they gain nothing from
// sharing the parent and benefit from independent t.Parallel() scheduling).
//
// GCS (service_account) is chosen over httpsource because httpsource provisions
// a Firehose stream asynchronously, which races with the framework's post-test
// DELETE and yields a flaky 500. GCS sources delete synchronously — no race, no
// retry helpers needed.
//
// Requires PANTHER_GCS_SA_* env vars (same as gcssource_resource_test.go) — the
// live lifecycle and malformed-import scenarios skip if they're missing.

var (
	compositeIDRegex     = regexp.MustCompile(`^[0-9a-f-]+/SOURCE_NO_DATA$`)
	thresholdBoundsRegex = regexp.MustCompile(`between 15 and 43200`)
	invalidTypeRegex     = regexp.MustCompile(`must be one of: \["SOURCE_NO_DATA"]`)
	invalidImportIDRegex = regexp.MustCompile(`Invalid Import ID`)
	notFoundSourceRegex  = regexp.MustCompile(`log source was not found|404`)
)

// TestLogSourceAlarmResource creates one parent gcssource (service_account) and
// exercises every live scenario against it in a single TestCase: CRUD lifecycle,
// valid import, three malformed-import variants, lower/upper threshold boundaries,
// and out-of-band-delete drift detection. Framework cleanup at the end destroys
// both the alarm (404-tolerant) and the gcssource (synchronous delete).
func TestLogSourceAlarmResource(t *testing.T) {
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

	parentLabel := strings.ReplaceAll(uuid.NewString(), "-", "")
	mkConfig := func(threshold int64) string {
		return providerConfig + testLogSourceAlarmConfig(parentLabel, credentials, projectId, subscriptionId, bucket, threshold)
	}

	malformedIDs := []struct{ name, id string }{
		{"no_separator", "just-a-uuid"},
		{"empty_source_id", "/SOURCE_NO_DATA"},
		{"empty_type", "41ed10a4-7791-460a-80b7-c0178baa3595/"},
	}

	steps := []resource.TestStep{
		// Step 1: Create parent gcssource + alarm at threshold 60.
		{
			PreConfig: func() {
				t.Log("Step 1: Create parent gcssource + alarm (PUT /log-source-alarms/{sourceId}/SOURCE_NO_DATA)")
			},
			Config: mkConfig(60),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("panther_log_source_alarm.test", "type", "SOURCE_NO_DATA"),
				resource.TestCheckResourceAttr("panther_log_source_alarm.test", "minutes_threshold", "60"),
				resource.TestCheckResourceAttrSet("panther_log_source_alarm.test", "source_id"),
				// Composite id must be "{source_id}/SOURCE_NO_DATA" — verify shape, not just presence.
				resource.TestMatchResourceAttr("panther_log_source_alarm.test", "id", compositeIDRegex),
			),
		},
		// Step 2: Valid import by composite "{source_id}/SOURCE_NO_DATA".
		{
			PreConfig:         func() { t.Log("Step 2: Valid import by {source_id}/SOURCE_NO_DATA") },
			ResourceName:      "panther_log_source_alarm.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	}

	// Steps 3-5: Malformed import attempts — each fails with ExpectError, leaving
	// state unchanged for the subsequent real-update steps.
	for _, tc := range malformedIDs {
		tc := tc
		steps = append(steps, resource.TestStep{
			PreConfig:     func() { t.Logf("Malformed import: %s (id=%q)", tc.name, tc.id) },
			ResourceName:  "panther_log_source_alarm.test",
			ImportState:   true,
			ImportStateId: tc.id,
			Config:        mkConfig(60),
			ExpectError:   invalidImportIDRegex,
		})
	}

	steps = append(steps,
		// Step 6: Update 60 → 1440 (typical user update — day-scale threshold).
		resource.TestStep{
			PreConfig: func() { t.Log("Step 6: Update (60 → 1440 minutes)") },
			Config:    mkConfig(1440),
			Check:     resource.TestCheckResourceAttr("panther_log_source_alarm.test", "minutes_threshold", "1440"),
		},
		// Step 7: Lower boundary (15 min — service-layer minimum).
		resource.TestStep{
			PreConfig: func() { t.Log("Step 7: Update to lower boundary (15 minutes)") },
			Config:    mkConfig(15),
			Check:     resource.TestCheckResourceAttr("panther_log_source_alarm.test", "minutes_threshold", "15"),
		},
		// Step 8: Upper boundary (43200 min / 30 days — service-layer maximum).
		resource.TestStep{
			PreConfig: func() { t.Log("Step 8: Update to upper boundary (43200 minutes / 30 days)") },
			Config:    mkConfig(43200),
			Check:     resource.TestCheckResourceAttr("panther_log_source_alarm.test", "minutes_threshold", "43200"),
		},
		// Step 9: Drift detection — manually delete the alarm via the REST API, then
		// verify Read detects 404 (handleReadError → RemoveResource). Framework cleanup
		// then calls Delete on both resources: alarm returns 404 (success via
		// handleDeleteError); gcssource deletes synchronously.
		resource.TestStep{
			PreConfig:          func() { t.Log("Step 9: Drift detection (out-of-band DELETE → expect non-empty plan)") },
			Config:             mkConfig(43200),
			Check:              manuallyDeleteLogSourceAlarm(t),
			ExpectNonEmptyPlan: true,
		},
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

// TestLogSourceAlarmResource_InvalidThreshold verifies the Between(15, 43200) validator
// rejects just-below (14) and just-above (43201) — the off-by-one edges. Plan-time only,
// no API call, no GCS creds required.
func TestLogSourceAlarmResource_InvalidThreshold(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		threshold int64
	}{
		{"14_rejected", 14},
		{"43201_rejected", 43201},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      providerConfig + testLogSourceAlarmStandaloneConfig("00000000-0000-0000-0000-000000000000", tc.threshold),
						ExpectError: thresholdBoundsRegex,
					},
				},
			})
		})
	}
}

// TestLogSourceAlarmResource_InvalidType verifies the stringvalidator.OneOf rejects alarm
// types other than SOURCE_NO_DATA at plan time. No API call, no GCS creds required.
func TestLogSourceAlarmResource_InvalidType(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "panther_log_source_alarm" "test" {
  source_id         = "00000000-0000-0000-0000-000000000000"
  type              = "SOURCE_PERMISSIONS_CHECKS"
  minutes_threshold = 60
}
`,
				ExpectError: invalidTypeRegex,
			},
		},
	})
}

// TestLogSourceAlarmResource_NonexistentSource verifies the API's pre-flight 404 (when the
// sourceId doesn't resolve to a real log source) bubbles through handleCreateError as an
// actionable diagnostic. Exercises the default branch of handleCreateError (neither
// 401/403 auth nor 409 conflict). No parent needed — the bogus sourceId can't collide
// with any real integration.
func TestLogSourceAlarmResource_NonexistentSource(t *testing.T) {
	t.Parallel()
	// Syntactically valid UUIDv4 that (almost certainly) doesn't resolve to any real
	// source in the target env. Collision probability is ~1 in 2^122.
	bogusSourceID := "ffffffff-ffff-4fff-bfff-ffffffffffff"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + testLogSourceAlarmStandaloneConfig(bogusSourceID, 60),
				ExpectError: notFoundSourceRegex,
			},
		},
	})
}

// manuallyDeleteLogSourceAlarm bypasses Terraform and deletes via the REST API directly,
// simulating out-of-band deletion for drift detection testing.
func manuallyDeleteLogSourceAlarm(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["panther_log_source_alarm.test"]
		if !ok {
			return fmt.Errorf("not found: panther_log_source_alarm.test")
		}
		if rs.Primary.ID == "" {
			return errors.New("alarm ID is not set")
		}
		// rs.Primary.ID is the composite "{source_id}/SOURCE_NO_DATA", exactly the
		// path suffix the API expects — concat and DELETE.
		c := client.NewRESTClient(os.Getenv("PANTHER_API_URL"), os.Getenv("PANTHER_API_TOKEN"))
		if err := client.RestDelete(context.Background(), c, logSourceAlarmPath+"/"+rs.Primary.ID); err != nil {
			return fmt.Errorf("could not delete alarm: %w", err)
		}
		t.Logf("Manually deleted alarm %s for drift detection test", rs.Primary.ID)
		return nil
	}
}

// testLogSourceAlarmConfig builds HCL with a panther_gcssource parent + the alarm attached.
// Uses service_account credentials — the credentials JSON is embedded via %q so it's
// properly quoted as a Terraform string literal.
func testLogSourceAlarmConfig(parentLabel, credentials, projectId, subscriptionId, bucket string, threshold int64) string {
	return fmt.Sprintf(`
resource "panther_gcssource" "parent" {
  integration_label = %q
  subscription_id   = %q
  project_id        = %q
  gcs_bucket        = %q
  credentials       = %q
  credentials_type  = "service_account"
  log_stream_type   = "Auto"
  prefix_log_types = [{
    prefix            = ""
    log_types         = ["GCP.AuditLog"]
    excluded_prefixes = []
  }]
}

resource "panther_log_source_alarm" "test" {
  source_id         = panther_gcssource.parent.id
  type              = "SOURCE_NO_DATA"
  minutes_threshold = %d
}
`, parentLabel, subscriptionId, projectId, bucket, credentials, threshold)
}

// testLogSourceAlarmStandaloneConfig builds a config that references a hard-coded source_id
// — useful for plan-time validator tests and no-parent API-error tests where no parent
// resource needs to exist.
func testLogSourceAlarmStandaloneConfig(sourceId string, threshold int64) string {
	return fmt.Sprintf(`
resource "panther_log_source_alarm" "test" {
  source_id         = "%s"
  type              = "SOURCE_NO_DATA"
  minutes_threshold = %d
}
`, sourceId, threshold)
}
