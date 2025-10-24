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

	// Rule management
	CreateRule(ctx context.Context, input CreateRuleInput) (Rule, error)
	UpdateRule(ctx context.Context, input UpdateRuleInput) (Rule, error)
	GetRule(ctx context.Context, id string) (Rule, error)
	DeleteRule(ctx context.Context, id string) error

	// Policy management
	CreatePolicy(ctx context.Context, input CreatePolicyInput) (Policy, error)
	UpdatePolicy(ctx context.Context, input UpdatePolicyInput) (Policy, error)
	GetPolicy(ctx context.Context, id string) (Policy, error)
	DeletePolicy(ctx context.Context, id string) error

	// Scheduled rule management
	CreateScheduledRule(ctx context.Context, input CreateScheduledRuleInput) (ScheduledRule, error)
	UpdateScheduledRule(ctx context.Context, input UpdateScheduledRuleInput) (ScheduledRule, error)
	GetScheduledRule(ctx context.Context, id string) (ScheduledRule, error)
	DeleteScheduledRule(ctx context.Context, id string) error

	// Simple rule management
	CreateSimpleRule(ctx context.Context, input CreateSimpleRuleInput) (SimpleRule, error)
	UpdateSimpleRule(ctx context.Context, input UpdateSimpleRuleInput) (SimpleRule, error)
	GetSimpleRule(ctx context.Context, id string) (SimpleRule, error)
	DeleteSimpleRule(ctx context.Context, id string) error
}

// CreateS3SourceInput Input for the createS3LogSource mutation
type CreateS3SourceInput struct {
	AwsAccountID               string                  `json:"awsAccountId"`
	KmsKey                     string                  `json:"kmsKey"`
	Label                      string                  `json:"label"`
	LogProcessingRole          string                  `json:"logProcessingRole"`
	LogStreamType              string                  `json:"logStreamType"`
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
	JsonArrayEnvelopeField string `json:"jsonArrayEnvelopeField,omitempty"`
	XmlRootElement         string `json:"xmlRootElement,omitempty"`
}

// HttpSourceModifiableAttributes attributes that can be modified on an http log source
type HttpSourceModifiableAttributes struct {
	IntegrationLabel     string
	LogStreamType        string
	LogTypes             []string
	LogStreamTypeOptions *LogStreamTypeOptions
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

// Rule types
type Rule struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	RuleModifiableAttributes
}

type RuleModifiableAttributes struct {
	DisplayName        string   `json:"displayName"`
	Body               string   `json:"body"`
	Description        string   `json:"description,omitempty"`
	Severity           string   `json:"severity,omitempty"`
	LogTypes           []string `json:"logTypes,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	References         []string `json:"references,omitempty"`
	Runbook            string   `json:"runbook,omitempty"`
	DedupPeriodMinutes int      `json:"dedupPeriodMinutes,omitempty"`
	Enabled            bool     `json:"enabled,omitempty"`
}

type CreateRuleInput struct {
	ID string `json:"id"`
	RuleModifiableAttributes
}

type UpdateRuleInput struct {
	ID string `json:"id"`
	RuleModifiableAttributes
}

// Policy types
type Policy struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"lastModified"`
	Version   string `json:"version"`
	PolicyModifiableAttributes
}

type PolicyModifiableAttributes struct {
	DisplayName   string   `json:"displayName"`
	Body          string   `json:"body"`
	Description   string   `json:"description,omitempty"`
	Severity      string   `json:"severity,omitempty"`
	ResourceTypes []string `json:"resourceTypes,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Runbook       string   `json:"runbook,omitempty"`
	Enabled       bool     `json:"enabled,omitempty"`
}

type CreatePolicyInput struct {
	ID string `json:"id"`
	PolicyModifiableAttributes
}

type UpdatePolicyInput struct {
	ID string `json:"id"`
	PolicyModifiableAttributes
}

// Scheduled rule types
type ScheduledRule struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"lastModified"`
	ScheduledRuleModifiableAttributes
}

type ScheduledRuleModifiableAttributes struct {
	DisplayName         string              `json:"displayName,omitempty"`
	Body                string              `json:"body"`
	Description         string              `json:"description,omitempty"`
	Severity            string              `json:"severity"`
	ScheduledQueries    []string            `json:"scheduledQueries,omitempty"`
	Tags                []string            `json:"tags,omitempty"`
	Runbook             string              `json:"runbook,omitempty"`
	DedupPeriodMinutes  int                 `json:"dedupPeriodMinutes,omitempty"`
	Enabled             bool                `json:"enabled,omitempty"`
	OutputIds           []string            `json:"outputIDs,omitempty"`
	Reports             map[string][]string `json:"reports,omitempty"`
	SummaryAttributes   []string            `json:"summaryAttributes,omitempty"`
	Threshold           int                 `json:"threshold,omitempty"`
	Managed             bool                `json:"managed,omitempty"`
}

type CreateScheduledRuleInput struct {
	ID string `json:"id"`
	ScheduledRuleModifiableAttributes
}

type UpdateScheduledRuleInput struct {
	ID string `json:"id"`
	ScheduledRuleModifiableAttributes
}

// Simple rule types
type SimpleRule struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"lastModified"`
	SimpleRuleModifiableAttributes
}

type SimpleRuleModifiableAttributes struct {
	DisplayName        string              `json:"displayName,omitempty"`
	Detection          string              `json:"detection"`
	Description        string              `json:"description,omitempty"`
	Severity           string              `json:"severity"`
	LogTypes           []string            `json:"logTypes,omitempty"`
	Tags               []string            `json:"tags,omitempty"`
	Runbook            string              `json:"runbook,omitempty"`
	DedupPeriodMinutes int                 `json:"dedupPeriodMinutes,omitempty"`
	Enabled            bool                `json:"enabled,omitempty"`
	AlertTitle         string              `json:"alertTitle,omitempty"`
	AlertContext       string              `json:"alertContext,omitempty"`
	DynamicSeverities  string              `json:"dynamicSeverities,omitempty"`
	GroupBy            string              `json:"groupBy,omitempty"`
	InlineFilters      string              `json:"inlineFilters,omitempty"`
	OutputIds          []string            `json:"outputIDs,omitempty"`
	PythonBody         string              `json:"pythonBody,omitempty"`
	Reports            map[string][]string `json:"reports,omitempty"`
	SummaryAttributes  []string            `json:"summaryAttributes,omitempty"`
	Threshold          int                 `json:"threshold,omitempty"`
	Managed            bool                `json:"managed,omitempty"`
}

type CreateSimpleRuleInput struct {
	ID string `json:"id"`
	SimpleRuleModifiableAttributes
}

type UpdateSimpleRuleInput struct {
	ID string `json:"id"`
	SimpleRuleModifiableAttributes
}
