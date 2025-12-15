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

package panther

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hasura/go-graphql-client"
	"io"
	"net/http"
	"strings"
	"terraform-provider-panther/internal/client"
)

const GraphqlPath = "/public/graphql"
const RestHttpSourcePath = "/log-sources/http"

var _ client.GraphQLClient = (*GraphQLClient)(nil)

var _ client.RestClient = (*RestClient)(nil)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type APIClient struct {
	*GraphQLClient
	*RestClient
}

type GraphQLClient struct {
	*graphql.Client
}

type RestClient struct {
	url string
	Doer
}

func NewGraphQLClient(url, token string) *GraphQLClient {
	return &GraphQLClient{
		graphql.NewClient(
			fmt.Sprintf("%s%s", url, GraphqlPath),
			NewAuthorizedHTTPClient(token)),
	}
}

func NewRestClient(url, token string) *RestClient {
	return &RestClient{
		url:  fmt.Sprintf("%s%s", url, RestHttpSourcePath),
		Doer: NewAuthorizedHTTPClient(token),
	}
}

func NewAPIClient(graphClient *GraphQLClient, restClient *RestClient) *APIClient {
	return &APIClient{
		graphClient,
		restClient,
	}
}

func CreateAPIClient(url, token string) *APIClient {
	// url in previous versions was provided including graphql endpoint,
	// we strip it here to keep it backwards compatible
	pantherUrl := strings.TrimSuffix(url, GraphqlPath)
	graphClient := NewGraphQLClient(pantherUrl, token)
	restClient := NewRestClient(pantherUrl, token)

	return NewAPIClient(graphClient, restClient)
}

func (c *RestClient) CreateHttpSource(ctx context.Context, input client.CreateHttpSourceInput) (client.HttpSource, error) {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("error marshaling data: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(jsonData))
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return client.HttpSource{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.HttpSource
	if err = json.Unmarshal(body, &response); err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) UpdateHttpSource(ctx context.Context, input client.UpdateHttpSourceInput) (client.HttpSource, error) {
	reqURL := fmt.Sprintf("%s/%s", c.url, input.IntegrationId)
	jsonData, err := json.Marshal(input)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("error marshaling data: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return client.HttpSource{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.HttpSource
	if err = json.Unmarshal(body, &response); err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) GetHttpSource(ctx context.Context, id string) (client.HttpSource, error) {
	reqURL := fmt.Sprintf("%s/%s", c.url, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return client.HttpSource{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.HttpSource
	if err = json.Unmarshal(body, &response); err != nil {
		return client.HttpSource{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) DeleteHttpSource(ctx context.Context, id string) error {
	reqURL := fmt.Sprintf("%s/%s", c.url, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	return nil
}

func (c *GraphQLClient) UpdateS3Source(ctx context.Context, input client.UpdateS3SourceInput) (client.UpdateS3SourceOutput, error) {
	var m struct {
		UpdateS3Source struct {
			client.UpdateS3SourceOutput
		} `graphql:"updateS3Source(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("UpdateS3Source"))
	if err != nil {
		return client.UpdateS3SourceOutput{}, fmt.Errorf("GraphQL mutation failed: %v", err)
	}
	return m.UpdateS3Source.UpdateS3SourceOutput, nil
}

func (c *GraphQLClient) DeleteSource(ctx context.Context, input client.DeleteSourceInput) (client.DeleteSourceOutput, error) {
	var m struct {
		DeleteSource struct {
			client.DeleteSourceOutput
		} `graphql:"deleteSource(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("DeleteSource"))
	if err != nil {
		return client.DeleteSourceOutput{}, fmt.Errorf("GraphQL mutation failed: %v", err)
	}
	return m.DeleteSource.DeleteSourceOutput, nil
}

func (c *GraphQLClient) GetS3Source(ctx context.Context, id string) (*client.S3LogIntegration, error) {
	var q struct {
		Source struct {
			S3LogIntegration client.S3LogIntegration `graphql:"... on S3LogIntegration"`
		} `graphql:"source(id: $id)"`
	}

	err := c.Query(ctx, &q, map[string]interface{}{
		"id": graphql.ID(id),
	}, graphql.OperationName("Source"))
	if err != nil {
		return nil, fmt.Errorf("GraphQL query failed: %v", err)
	}
	return &q.Source.S3LogIntegration, nil
}

func (c *GraphQLClient) CreateS3Source(ctx context.Context, input client.CreateS3SourceInput) (client.CreateS3SourceOutput, error) {
	var m struct {
		CreateS3Source struct {
			client.CreateS3SourceOutput
		} `graphql:"createS3Source(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]any{
		"input": input,
	}, graphql.OperationName("CreateS3Source"))
	if err != nil {
		return client.CreateS3SourceOutput{}, fmt.Errorf("GraphQL mutation failed: %w", err)
	}
	return m.CreateS3Source.CreateS3SourceOutput, nil
}

func getErrorResponseMsg(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("failed to read response body: %s", err.Error())
	}

	var errResponse client.HttpErrorResponse
	if err = json.Unmarshal(body, &errResponse); err != nil {
		return fmt.Sprintf("failed to unmarshal response body to get error response: %s", err.Error())
	}

	return errResponse.Message
}

// Generic REST helper for Rule endpoints
func (c *RestClient) doRuleRequest(ctx context.Context, method, path string, input interface{}, expectedStatus int) ([]byte, error) {
	// Extract base URL without the /log-sources/http suffix
	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	fullURL := fmt.Sprintf("%s%s", baseURL, path)

	var body io.Reader
	if input != nil {
		jsonData, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("error marshaling data: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return nil, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return responseBody, nil
}

// Rule methods
func (c *RestClient) CreateRule(ctx context.Context, input client.CreateRuleInput) (client.Rule, error) {
	body, err := c.doRuleRequest(ctx, http.MethodPost, "/rules", input, http.StatusOK)
	if err != nil {
		return client.Rule{}, err
	}

	var response client.Rule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Rule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) UpdateRule(ctx context.Context, input client.UpdateRuleInput) (client.Rule, error) {
	path := fmt.Sprintf("/rules/%s", input.ID)
	body, err := c.doRuleRequest(ctx, http.MethodPut, path, input, http.StatusOK)
	if err != nil {
		return client.Rule{}, err
	}

	var response client.Rule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Rule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) GetRule(ctx context.Context, id string) (client.Rule, error) {
	path := fmt.Sprintf("/rules/%s", id)
	body, err := c.doRuleRequest(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return client.Rule{}, err
	}

	var response client.Rule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Rule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) DeleteRule(ctx context.Context, id string) error {
	path := fmt.Sprintf("/rules/%s", id)
	_, err := c.doRuleRequest(ctx, http.MethodDelete, path, nil, http.StatusNoContent)
	return err
}

// Policy methods
func (c *RestClient) CreatePolicy(ctx context.Context, input client.CreatePolicyInput) (client.Policy, error) {
	body, err := c.doRuleRequest(ctx, http.MethodPost, "/policies", input, http.StatusOK)
	if err != nil {
		return client.Policy{}, err
	}

	var response client.Policy
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Policy{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) UpdatePolicy(ctx context.Context, input client.UpdatePolicyInput) (client.Policy, error) {
	path := fmt.Sprintf("/policies/%s", input.ID)
	body, err := c.doRuleRequest(ctx, http.MethodPut, path, input, http.StatusOK)
	if err != nil {
		return client.Policy{}, err
	}

	var response client.Policy
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Policy{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) GetPolicy(ctx context.Context, id string) (client.Policy, error) {
	path := fmt.Sprintf("/policies/%s", id)
	body, err := c.doRuleRequest(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return client.Policy{}, err
	}

	var response client.Policy
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Policy{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) DeletePolicy(ctx context.Context, id string) error {
	path := fmt.Sprintf("/policies/%s", id)
	_, err := c.doRuleRequest(ctx, http.MethodDelete, path, nil, http.StatusNoContent)
	return err
}

// Scheduled rule methods
func (c *RestClient) CreateScheduledRule(ctx context.Context, input client.CreateScheduledRuleInput) (client.ScheduledRule, error) {
	body, err := c.doRuleRequest(ctx, http.MethodPost, "/scheduled-rules", input, http.StatusOK)
	if err != nil {
		return client.ScheduledRule{}, err
	}

	var response client.ScheduledRule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.ScheduledRule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) UpdateScheduledRule(ctx context.Context, input client.UpdateScheduledRuleInput) (client.ScheduledRule, error) {
	path := fmt.Sprintf("/scheduled-rules/%s", input.ID)
	body, err := c.doRuleRequest(ctx, http.MethodPut, path, input, http.StatusOK)
	if err != nil {
		return client.ScheduledRule{}, err
	}

	var response client.ScheduledRule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.ScheduledRule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) GetScheduledRule(ctx context.Context, id string) (client.ScheduledRule, error) {
	path := fmt.Sprintf("/scheduled-rules/%s", id)
	body, err := c.doRuleRequest(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return client.ScheduledRule{}, err
	}

	var response client.ScheduledRule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.ScheduledRule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) DeleteScheduledRule(ctx context.Context, id string) error {
	path := fmt.Sprintf("/scheduled-rules/%s", id)
	_, err := c.doRuleRequest(ctx, http.MethodDelete, path, nil, http.StatusNoContent)
	return err
}

// Simple rule methods
func (c *RestClient) CreateSimpleRule(ctx context.Context, input client.CreateSimpleRuleInput) (client.SimpleRule, error) {
	body, err := c.doRuleRequest(ctx, http.MethodPost, "/simple-rules", input, http.StatusOK)
	if err != nil {
		return client.SimpleRule{}, err
	}

	var response client.SimpleRule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.SimpleRule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) UpdateSimpleRule(ctx context.Context, input client.UpdateSimpleRuleInput) (client.SimpleRule, error) {
	path := fmt.Sprintf("/simple-rules/%s", input.ID)
	body, err := c.doRuleRequest(ctx, http.MethodPut, path, input, http.StatusOK)
	if err != nil {
		return client.SimpleRule{}, err
	}

	var response client.SimpleRule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.SimpleRule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) GetSimpleRule(ctx context.Context, id string) (client.SimpleRule, error) {
	path := fmt.Sprintf("/simple-rules/%s", id)
	body, err := c.doRuleRequest(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return client.SimpleRule{}, err
	}

	var response client.SimpleRule
	if err = json.Unmarshal(body, &response); err != nil {
		return client.SimpleRule{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) DeleteSimpleRule(ctx context.Context, id string) error {
	path := fmt.Sprintf("/simple-rules/%s", id)
	_, err := c.doRuleRequest(ctx, http.MethodDelete, path, nil, http.StatusNoContent)
	return err
}
