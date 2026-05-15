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

// AwsScanConfig is the nested scan configuration for a cloud account integration.
type AwsScanConfig struct {
	AuditRole string `json:"auditRole"`
}

// AwsCloudAccountInput is the POST/PUT body. AwsAccountId has `omitempty` so
// PUT (which drops it) doesn't send the zero value; the exclusion lists
// deliberately don't, so cleared lists serialize as `[]` on the wire.
type AwsCloudAccountInput struct {
	IntegrationLabel        string        `json:"integrationLabel"`
	AwsAccountId            string        `json:"awsAccountId,omitempty"`
	AwsScanConfig           AwsScanConfig `json:"awsScanConfig"`
	RegionIgnoreList        []string      `json:"regionIgnoreList"`
	ResourceTypeIgnoreList  []string      `json:"resourceTypeIgnoreList"`
	ResourceRegexIgnoreList []string      `json:"resourceRegexIgnoreList"`
}

// AwsCloudAccount is the response body. The REST API guarantees non-null `[]`
// for the three exclusion lists, so they're typed as plain slices.
type AwsCloudAccount struct {
	IntegrationId string `json:"integrationId"`
	AwsCloudAccountInput
}
