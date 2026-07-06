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

// SimpleRuleInput is the request body for creating or updating a SimpleRule.
type SimpleRuleInput struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"displayName,omitempty"`
	Detection          string   `json:"detection"`
	Description        string   `json:"description,omitempty"`
	Severity           string   `json:"severity"`
	LogTypes           []string `json:"logTypes,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	Runbook            string   `json:"runbook,omitempty"`
	DedupPeriodMinutes int      `json:"dedupPeriodMinutes,omitempty"`
	Enabled            bool     `json:"enabled,omitempty"`
	AlertTitle         string   `json:"alertTitle,omitempty"`
	AlertContext       string   `json:"alertContext,omitempty"`
	DynamicSeverities  string   `json:"dynamicSeverities,omitempty"`
	GroupBy            string   `json:"groupBy,omitempty"`
	InlineFilters      string   `json:"inlineFilters,omitempty"`
	PythonBody         string   `json:"pythonBody,omitempty"`
	Threshold          int      `json:"threshold,omitempty"`
}

// SimpleRule is the API response (embeds SimpleRuleInput plus server-managed fields).
type SimpleRule struct {
	CreatedAt    string `json:"createdAt"`
	LastModified string `json:"lastModified"`
	SimpleRuleInput
}
