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

type HttpSource struct {
	IntegrationId string
	HttpSourceModifiableAttributes
}

// HttpLogStreamTypeOptions contains options specific to the log stream type for HTTP sources
type HttpLogStreamTypeOptions struct {
	JsonArrayEnvelopeField string `json:"jsonArrayEnvelopeField,omitempty"`
	XmlRootElement         string `json:"xmlRootElement,omitempty"`
}

// HttpSourceModifiableAttributes attributes that can be modified on an http log source
type HttpSourceModifiableAttributes struct {
	IntegrationLabel     string
	LogStreamType        string
	LogTypes             []string
	LogStreamTypeOptions *HttpLogStreamTypeOptions
	AuthHmacAlg          string
	AuthHeaderKey        string
	AuthPassword         string
	AuthSecretValue      string
	AuthMethod           string
	AuthUsername         string
	AuthBearerToken      string
}

// CreateHttpSourceInput Input for creating an http log source
type CreateHttpSourceInput struct {
	HttpSourceModifiableAttributes
}

// UpdateHttpSourceInput input for updating an http log source
type UpdateHttpSourceInput struct {
	IntegrationId string
	HttpSourceModifiableAttributes
}
