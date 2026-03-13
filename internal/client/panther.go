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

import (
	"context"
)

type GraphQLClient interface {
	CreateS3Source(ctx context.Context, input CreateS3SourceInput) (CreateS3SourceOutput, error)
	UpdateS3Source(ctx context.Context, input UpdateS3SourceInput) (UpdateS3SourceOutput, error)
	GetS3Source(ctx context.Context, id string) (*S3LogIntegration, error)
	DeleteSource(ctx context.Context, input DeleteSourceInput) (DeleteSourceOutput, error)
}

type RestClient interface {
	CreateHttpSource(ctx context.Context, input CreateHttpSourceInput) (HttpSource, error)
	UpdateHttpSource(ctx context.Context, input UpdateHttpSourceInput) (HttpSource, error)
	GetHttpSource(ctx context.Context, id string) (HttpSource, error)
	DeleteHttpSource(ctx context.Context, id string) error

	CreatePubSubSource(ctx context.Context, input CreatePubSubSourceInput) (PubSubSource, error)
	UpdatePubSubSource(ctx context.Context, input UpdatePubSubSourceInput) (PubSubSource, error)
	GetPubSubSource(ctx context.Context, id string) (PubSubSource, error)
	DeletePubSubSource(ctx context.Context, id string) error
}

// CreateS3SourceInput Input for the createS3LogSource mutation
type CreateS3SourceInput struct {
	AwsAccountID               string                  `json:"awsAccountId"`
	KmsKey                     string                  `json:"kmsKey"`
	Label                      string                  `json:"label"`
	LogProcessingRole          string                  `json:"logProcessingRole"`
	LogStreamType              string                  `json:"logStreamType"`
	LogStreamTypeOptions       *LogStreamTypeOptions   `json:"logStreamTypeOptions,omitempty"`
	ManagedBucketNotifications bool                    `json:"managedBucketNotifications"`
	S3Bucket                   string                  `json:"s3Bucket"`
	S3PrefixLogTypes           []S3PrefixLogTypesInput `json:"s3PrefixLogTypes"`
}

// CreateS3SourceOutput output for the createS3LogSource mutation
type CreateS3SourceOutput struct {
	LogSource *S3LogIntegration `graphql:"logSource"`
}

// UpdateS3SourceInput input for the updateS3Source mutation
type UpdateS3SourceInput struct {
	ID                         string                  `json:"id"`
	KmsKey                     string                  `json:"kmsKey"`
	Label                      string                  `json:"label"`
	LogProcessingRole          string                  `json:"logProcessingRole"`
	LogStreamType              string                  `json:"logStreamType"`
	LogStreamTypeOptions       *LogStreamTypeOptions   `json:"logStreamTypeOptions,omitempty"`
	ManagedBucketNotifications bool                    `json:"managedBucketNotifications"`
	S3PrefixLogTypes           []S3PrefixLogTypesInput `json:"s3PrefixLogTypes"`
}

// UpdateS3SourceOutput output for the updateS3LogSource mutation
type UpdateS3SourceOutput struct {
	LogSource *S3LogIntegration `graphql:"logSource"`
}

// DeleteSourceInput input for the deleteSource mutation
type DeleteSourceInput struct {
	ID string `json:"id"`
}

// DeleteSourceOutput output for the deleteSource mutation
type DeleteSourceOutput struct {
	ID string `json:"id"`
}

// S3LogIntegration Represents an S3 Log Source Integration
type S3LogIntegration struct {
	// The ID of the AWS Account where the S3 Bucket is located
	AwsAccountID string `graphql:"awsAccountId"`
	// The ID of the Log Source integration
	IntegrationID string `graphql:"integrationId"`
	// The name of the Log Source integration
	IntegrationLabel string `graphql:"integrationLabel"`
	// The type of Log Source integration
	IntegrationType string `graphql:"integrationType"`
	// True if the Log Source can be modified
	IsEditable bool `graphql:"isEditable"`
	// KMS key used to access the S3 Bucket
	KmsKey string `graphql:"kmsKey"`
	// The AWS Role used to access the S3 Bucket
	LogProcessingRole *string `graphql:"logProcessingRole"`
	// The format of the log files being ingested
	LogStreamType *string `graphql:"logStreamType"`
	// Options specific to the log stream type
	LogStreamTypeOptions *LogStreamTypeOptions `graphql:"logStreamTypeOptions"`
	// True if bucket notifications are being managed by Panther
	ManagedBucketNotifications bool `json:"managedBucketNotifications"`
	// The S3 Bucket name being ingested
	S3Bucket string `graphql:"s3Bucket"`
	// The prefix on the S3 Bucket name being ingested
	S3Prefix *string `graphql:"s3Prefix"`
	// Used to map prefixes to log types
	S3PrefixLogTypes []S3PrefixLogTypes `graphql:"s3PrefixLogTypes"`
}

// S3PrefixLogTypesInput Mapping of S3 prefixes to log types
type S3PrefixLogTypesInput struct {
	// S3 Prefixes to exclude
	ExcludedPrefixes []string `json:"excludedPrefixes"`
	// Log types to map to prefix
	LogTypes []string `json:"logTypes"`
	// S3 Prefix to map to log types
	Prefix string `json:"prefix"`
}

type S3PrefixLogTypes struct {
	// S3 Prefixes to exclude
	ExcludedPrefixes []string `graphql:"excludedPrefixes"`
	// Log types to map to prefix
	LogTypes []string `graphql:"logTypes"`
	// S3 Prefix to map to log types
	Prefix string `graphql:"prefix"`
}

type HttpSource struct {
	IntegrationId string
	HttpSourceModifiableAttributes
}

// LogStreamTypeOptions contains options specific to the log stream type
type LogStreamTypeOptions struct {
	JsonArrayEnvelopeField *string `json:"jsonArrayEnvelopeField,omitempty"`
	RetainEnvelopeFields   *bool   `json:"retainEnvelopeFields,omitempty"`
	XmlRootElement         *string `json:"xmlRootElement,omitempty"`
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

type HttpErrorResponse struct {
	Message string
}

// PubSubSource represents a GCP Pub/Sub log source integration (API response)
type PubSubSource struct {
	IntegrationId   string `json:"integrationId"`
	CredentialsType string `json:"credentialsType"`
	PubSubSourceModifiableAttributes
}

// PubSubSourceModifiableAttributes attributes that can be modified on a Pub/Sub log source
type PubSubSourceModifiableAttributes struct {
	IntegrationLabel         string                      `json:"integrationLabel"`
	SubscriptionId           string                      `json:"subscriptionId"`
	ProjectId                string                      `json:"projectId"`
	Credentials              string                      `json:"credentials,omitempty"`
	LogTypes                 []string                    `json:"logTypes"`
	LogStreamType            string                      `json:"logStreamType"`
	LogStreamTypeOptions     *PubSubLogStreamTypeOptions `json:"logStreamTypeOptions,omitempty"`
	EnforcedRegionalEndpoint string                      `json:"enforcedRegionalEndpoint,omitempty"`
}

// PubSubLogStreamTypeOptions contains options specific to the log stream type for Pub/Sub sources
type PubSubLogStreamTypeOptions struct {
	JsonArrayEnvelopeField string `json:"jsonArrayEnvelopeField,omitempty"`
	XmlRootElement         string `json:"xmlRootElement,omitempty"`
}

// CreatePubSubSourceInput input for creating a Pub/Sub log source
type CreatePubSubSourceInput struct {
	PubSubSourceModifiableAttributes
}

// UpdatePubSubSourceInput input for updating a Pub/Sub log source
type UpdatePubSubSourceInput struct {
	IntegrationId string `json:"-"`
	PubSubSourceModifiableAttributes
}
