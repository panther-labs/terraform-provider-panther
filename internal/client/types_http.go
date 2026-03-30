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

// HttpSource represents an HTTP log source integration (API response).
type HttpSource struct {
	IntegrationId string `json:"integrationId"`
	HttpSourceInput
}

// HttpSourceInput is the request body for creating or updating an HTTP log source.
type HttpSourceInput struct {
	IntegrationLabel     string                    `json:"integrationLabel"`
	LogStreamType        string                    `json:"logStreamType"`
	LogTypes             []string                  `json:"logTypes"`
	LogStreamTypeOptions *HttpLogStreamTypeOptions `json:"logStreamTypeOptions,omitempty"`
	AuthHmacAlg          string                    `json:"authHmacAlg,omitempty"`
	AuthHeaderKey        string                    `json:"authHeaderKey,omitempty"`
	AuthPassword         string                    `json:"authPassword,omitempty"`
	AuthSecretValue      string                    `json:"authSecretValue,omitempty"`
	AuthMethod           string                    `json:"authMethod"`
	AuthUsername         string                    `json:"authUsername,omitempty"`
	AuthBearerToken      string                    `json:"authBearerToken,omitempty"`
}

// HttpLogStreamTypeOptions contains options specific to the log stream type for HTTP sources.
type HttpLogStreamTypeOptions struct {
	JsonArrayEnvelopeField string `json:"jsonArrayEnvelopeField,omitempty"`
	XmlRootElement         string `json:"xmlRootElement,omitempty"`
}
