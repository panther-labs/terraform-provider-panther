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

// GcsSource represents a GCS log source integration (API response).
type GcsSource struct {
	IntegrationId string `json:"integrationId"`
	GcsSourceInput
}

// GcsSourceInput is the request body for creating or updating a GCS log source.
type GcsSourceInput struct {
	IntegrationLabel     string                   `json:"integrationLabel"`
	SubscriptionId       string                   `json:"subscriptionId"`
	ProjectId            string                   `json:"projectId,omitempty"`
	GcsBucket            string                   `json:"gcsBucket"`
	Credentials          string                   `json:"credentials,omitempty"`
	CredentialsType      string                   `json:"credentialsType"`
	LogStreamType        string                   `json:"logStreamType"`
	LogStreamTypeOptions *GcsLogStreamTypeOptions `json:"logStreamTypeOptions,omitempty"`
	PrefixLogTypes       []GcsPrefixLogTypesInput `json:"prefixLogTypes"`
}

// GcsLogStreamTypeOptions contains options specific to the log stream type for GCS sources.
type GcsLogStreamTypeOptions struct {
	JsonArrayEnvelopeField string `json:"jsonArrayEnvelopeField,omitempty"`
	XmlRootElement         string `json:"xmlRootElement,omitempty"`
}

// GcsPrefixLogTypesInput represents a prefix-to-log-types mapping for a GCS source.
type GcsPrefixLogTypesInput struct {
	Prefix           string   `json:"prefix"`
	LogTypes         []string `json:"logTypes"`
	ExcludedPrefixes []string `json:"excludedPrefixes"`
}
