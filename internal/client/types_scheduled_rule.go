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

package client

// ScheduledRuleInput is the request body for creating or updating a ScheduledRule.
type ScheduledRuleInput struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"displayName,omitempty"`
	Body               string   `json:"body"`
	Description        string   `json:"description,omitempty"`
	Severity           string   `json:"severity"`
	ScheduledQueries   []string `json:"scheduledQueries,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	Runbook            string   `json:"runbook,omitempty"`
	DedupPeriodMinutes int      `json:"dedupPeriodMinutes,omitempty"`
	Enabled            bool     `json:"enabled,omitempty"`
	Threshold          int      `json:"threshold,omitempty"`
}

// ScheduledRule is the API response (embeds ScheduledRuleInput plus server-managed fields).
type ScheduledRule struct {
	CreatedAt    string `json:"createdAt"`
	LastModified string `json:"lastModified"`
	ScheduledRuleInput
}
