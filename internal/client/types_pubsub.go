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

// PubSubSource represents a GCP Pub/Sub log source integration (API response).
type PubSubSource struct {
	IntegrationId string `json:"integrationId"`
	PubSubSourceInput
}

// PubSubSourceInput is the request body for creating or updating a Pub/Sub log source.
type PubSubSourceInput struct {
	IntegrationLabel     string                      `json:"integrationLabel"`
	SubscriptionId       string                      `json:"subscriptionId"`
	ProjectId            string                      `json:"projectId,omitempty"`
	Credentials          string                      `json:"credentials,omitempty"`
	CredentialsType      string                      `json:"credentialsType"`
	LogTypes             []string                    `json:"logTypes"`
	LogStreamType        string                      `json:"logStreamType"`
	LogStreamTypeOptions *PubSubLogStreamTypeOptions `json:"logStreamTypeOptions,omitempty"`
	RegionalEndpoint     string                      `json:"regionalEndpoint,omitempty"`
}

// PubSubLogStreamTypeOptions contains options specific to the log stream type for Pub/Sub sources.
type PubSubLogStreamTypeOptions struct {
	JsonArrayEnvelopeField string `json:"jsonArrayEnvelopeField,omitempty"`
	XmlRootElement         string `json:"xmlRootElement,omitempty"`
}
