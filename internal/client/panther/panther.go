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

func (c *RestClient) CreateUser(ctx context.Context, input client.CreateUserInput) (client.User, error) {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return client.User{}, fmt.Errorf("error marshaling data: %w", err)
	}

	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/users", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return client.User{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.User{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return client.User{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.User{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.User
	if err = json.Unmarshal(body, &response); err != nil {
		return client.User{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) UpdateUser(ctx context.Context, input client.UpdateUserInput) (client.User, error) {
	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/users/%s", baseURL, input.ID)
	jsonData, err := json.Marshal(input.UserModifiableAttributes)
	if err != nil {
		return client.User{}, fmt.Errorf("error marshaling data: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return client.User{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.User{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return client.User{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.User{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.User
	if err = json.Unmarshal(body, &response); err != nil {
		return client.User{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) GetUser(ctx context.Context, id string) (client.User, error) {
	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/users/%s", baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return client.User{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.User{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return client.User{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.User{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.User
	if err = json.Unmarshal(body, &response); err != nil {
		return client.User{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) DeleteUser(ctx context.Context, id string) error {
	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/users/%s", baseURL, id)
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

func (c *RestClient) CreateRole(ctx context.Context, input client.CreateRoleInput) (client.Role, error) {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return client.Role{}, fmt.Errorf("error marshaling data: %w", err)
	}

	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/roles", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return client.Role{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.Role
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Role{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) UpdateRole(ctx context.Context, input client.UpdateRoleInput) (client.Role, error) {
	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/roles/%s", baseURL, input.ID)
	jsonData, err := json.Marshal(input.RoleModifiableAttributes)
	if err != nil {
		return client.Role{}, fmt.Errorf("error marshaling data: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return client.Role{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.Role
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Role{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) GetRole(ctx context.Context, id string) (client.Role, error) {
	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/roles/%s", baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return client.Role{}, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, getErrorResponseMsg(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return client.Role{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var response client.Role
	if err = json.Unmarshal(body, &response); err != nil {
		return client.Role{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c *RestClient) DeleteRole(ctx context.Context, id string) error {
	baseURL := strings.TrimSuffix(c.url, RestHttpSourcePath)
	reqURL := fmt.Sprintf("%s/roles/%s", baseURL, id)
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
