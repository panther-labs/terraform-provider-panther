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
	"net/http"
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
// Architecture: the live lifecycle scenarios share a single parent log source
// inside ONE resource.TestCase so the parent is created and torn down once;
// plan-time tests and the no-parent 404 test stay as independent top-level
// tests with their own t.Parallel().
//
// Parent: panther_httpsource — cheapest to provision (only PANTHER_API_URL and
// PANTHER_API_TOKEN required). Its Kinesis Firehose is async, so a naive teardown
// races CREATING and returns 5xx; the final step OOB-deletes the parent through
// a retry loop, after which refresh observes 404 and drops both resources from
// state, leaving the framework's destroy phase a no-op.

var (
	compositeIDRegex     = regexp.MustCompile(`^[0-9a-f-]+/SOURCE_NO_DATA$`)
	thresholdBoundsRegex = regexp.MustCompile(`between 15 and 43200`)
	invalidTypeRegex     = regexp.MustCompile(`must be one of: \["SOURCE_NO_DATA"]`)
	invalidImportIDRegex = regexp.MustCompile(`Invalid Import ID`)
	notFoundSourceRegex  = regexp.MustCompile(`log source was not found|404`)
)

// TestLogSourceAlarmResource exercises CRUD, import (valid + four malformed
// variants), lower/upper threshold boundaries, out-of-band drift detection, and
// drift recovery against a shared parent — see the file-level docstring for the
// architecture and parent-choice rationale. CheckDestroy is wired for parity with
// checkS3SourceDestroyed; in the steady state of this test, refresh has already
// pruned every resource from state, so its loop body is unreachable.
func TestLogSourceAlarmResource(t *testing.T) {
	t.Parallel()

	parentLabel := strings.ReplaceAll(uuid.NewString(), "-", "")
	mkConfig := func(threshold int64) string {
		return providerConfig + testLogSourceAlarmConfig(parentLabel, threshold)
	}

	malformedIDs := []struct{ name, id string }{
		{"no_separator", "just-a-uuid"},
		{"empty_source_id", "/SOURCE_NO_DATA"},
		{"empty_type", "41ed10a4-7791-460a-80b7-c0178baa3595/"},
		{"multi_slash", "41ed10a4-7791-460a-80b7-c0178baa3595/SOURCE_NO_DATA/extra"},
	}

	steps := []resource.TestStep{
		{
			PreConfig: func() {
				t.Log("Step 1: Create parent httpsource + alarm (PUT /log-source-alarms/{sourceId}/SOURCE_NO_DATA)")
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
		// verify Read detects 404 (handleReadError → RemoveResource) and proposes
		// recreation in the next plan.
		resource.TestStep{
			PreConfig:          func() { t.Log("Step 9: Drift detection (out-of-band DELETE → expect non-empty plan)") },
			Config:             mkConfig(43200),
			Check:              manuallyDeleteLogSourceAlarm(t),
			ExpectNonEmptyPlan: true,
		},
		// Step 10: Re-apply the same config to recreate the alarm drifted away in
		// Step 9, verifying the Create path runs cleanly after a Read-detected 404
		// (drift recovery).
		resource.TestStep{
			PreConfig: func() { t.Log("Step 10: Recreate alarm post-drift (drift recovery)") },
			Config:    mkConfig(43200),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("panther_log_source_alarm.test", "id"),
				resource.TestCheckResourceAttr("panther_log_source_alarm.test", "minutes_threshold", "43200"),
			),
		},
		// Step 11: Manually delete the parent httpsource with bounded retries to
		// sidestep the post-create Kinesis Firehose race (DELETE returns 5xx while
		// the stream is CREATING). Subsequent refresh observes 404 on the parent
		// and on the now-orphaned alarm, dropping both from state — so the
		// framework's automatic destroy phase finds an empty state and is a no-op.
		// RefreshState skips a redundant plan/apply cycle: state from Step 10 is
		// reused, Check runs the OOB delete, and the post-Check plan picks up the
		// drift (ExpectNonEmptyPlan).
		resource.TestStep{
			PreConfig:          func() { t.Log("Step 11: Manually delete parent httpsource (Firehose race workaround)") },
			RefreshState:       true,
			Check:              manuallyDeleteSource(t, "panther_httpsource.parent", httpSourcePath),
			ExpectNonEmptyPlan: true,
		},
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkLogSourceAlarmDestroyed,
		Steps:                    steps,
	})
}

// TestLogSourceAlarmResource_InvalidThreshold verifies the Between(15, 43200) validator
// rejects just-below (14) and just-above (43201) — the off-by-one edges. Plan-time only,
// no API call.
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
// types other than SOURCE_NO_DATA at plan time. No API call.
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

// checkLogSourceAlarmDestroyed verifies that each panther_log_source_alarm tracked in
// the final test state has actually been removed from the Panther API — closes the
// silent-failure window where Delete returns no diagnostic but the alarm still exists
// remotely. Mirrors checkS3SourceDestroyed in s3source_resource_test.go.
func checkLogSourceAlarmDestroyed(s *terraform.State) error {
	c := client.NewRESTClient(os.Getenv("PANTHER_API_URL"), os.Getenv("PANTHER_API_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "panther_log_source_alarm" {
			continue
		}
		// rs.Primary.ID is the composite "{source_id}/SOURCE_NO_DATA", exactly the
		// path suffix the API expects.
		_, err := client.RestDo[client.LogSourceAlarm](context.Background(), c, http.MethodGet, logSourceAlarmPath+"/"+rs.Primary.ID, nil)
		if err == nil {
			return fmt.Errorf("alarm %s still exists after destroy", rs.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking alarm %s: %w", rs.Primary.ID, err)
		}
	}
	return nil
}

// testLogSourceAlarmConfig builds HCL with a panther_httpsource parent + the alarm
// attached. See the file-level docstring for why httpsource is the parent of choice.
func testLogSourceAlarmConfig(parentLabel string, threshold int64) string {
	return fmt.Sprintf(`
resource "panther_httpsource" "parent" {
  integration_label = %q
  log_stream_type   = "Auto"
  log_types         = ["AWS.CloudFrontAccess"]
  auth_method       = "SharedSecret"
  auth_header_key   = "x-api-key"
  auth_secret_value = "test-secret-value"
}

resource "panther_log_source_alarm" "test" {
  source_id         = panther_httpsource.parent.id
  type              = "SOURCE_NO_DATA"
  minutes_threshold = %d
}
`, parentLabel, threshold)
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
