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

// S3SourceCreateInput has awsAccountId because it's immutable after creation and excluded from PUT.
type S3SourceCreateInput struct {
	AwsAccountId               string                  `json:"awsAccountId"`
	IntegrationLabel           string                  `json:"integrationLabel"`
	S3Bucket                   string                  `json:"s3Bucket"`
	KmsKey                     string                  `json:"kmsKey,omitempty"`
	LogProcessingRole          string                  `json:"logProcessingRole"`
	LogStreamType              string                  `json:"logStreamType"`
	LogStreamTypeOptions       *S3LogStreamTypeOptions `json:"logStreamTypeOptions,omitempty"`
	ManagedBucketNotifications bool                    `json:"managedBucketNotifications"`
	S3PrefixLogTypes           []S3PrefixLogTypesInput `json:"s3PrefixLogTypes"`
}

// S3SourceUpdateInput excludes awsAccountId and s3Bucket because they're immutable after creation.
type S3SourceUpdateInput struct {
	IntegrationLabel           string                  `json:"integrationLabel"`
	KmsKey                     string                  `json:"kmsKey,omitempty"`
	LogProcessingRole          string                  `json:"logProcessingRole"`
	LogStreamType              string                  `json:"logStreamType"`
	LogStreamTypeOptions       *S3LogStreamTypeOptions `json:"logStreamTypeOptions,omitempty"`
	ManagedBucketNotifications bool                    `json:"managedBucketNotifications"`
	S3PrefixLogTypes           []S3PrefixLogTypesInput `json:"s3PrefixLogTypes"`
}

// S3Source is the REST API response for GET/POST/PUT /log-sources/s3.
type S3Source struct {
	IntegrationId              string                  `json:"integrationId"`
	IntegrationLabel           string                  `json:"integrationLabel"`
	AwsAccountId               string                  `json:"awsAccountId"`
	S3Bucket                   string                  `json:"s3Bucket"`
	KmsKey                     string                  `json:"kmsKey"`
	LogProcessingRole          string                  `json:"logProcessingRole"`
	LogStreamType              string                  `json:"logStreamType"`
	LogStreamTypeOptions       *S3LogStreamTypeOptions `json:"logStreamTypeOptions,omitempty"`
	ManagedBucketNotifications bool                    `json:"managedBucketNotifications"`
	S3PrefixLogTypes           []S3PrefixLogTypesInput `json:"s3PrefixLogTypes"`
}

// S3PrefixLogTypesInput represents a prefix-to-log-types mapping for an S3 source.
// Used for both request bodies and response deserialization.
type S3PrefixLogTypesInput struct {
	ExcludedPrefixes []string `json:"excludedPrefixes"`
	LogTypes         []string `json:"logTypes"`
	Prefix           string   `json:"prefix"`
}

type S3LogStreamTypeOptions struct {
	JsonArrayEnvelopeField string `json:"jsonArrayEnvelopeField,omitempty"`
	RetainEnvelopeFields   bool   `json:"retainEnvelopeFields,omitempty"`
	XmlRootElement         string `json:"xmlRootElement,omitempty"`
}
