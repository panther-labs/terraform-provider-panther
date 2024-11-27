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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hasura/go-graphql-client"
	"io"
	"net/http"
	"terraform-provider-panther/internal/client"
	"time"
)

var _ client.GraphQLClient = (*GraphQLClient)(nil)

var _ client.RestClient = (*RestClient)(nil)

type APIClient struct {
	*GraphQLClient
	*RestClient
}

type GraphQLClient struct {
	*graphql.Client
}

type RestClient struct {
	token string
	url   string
	*http.Client
}

func NewGraphQLClient(url, token string) *GraphQLClient {
	return &GraphQLClient{
		graphql.NewClient(
			url,
			NewAuthorizedHTTPClient(token)),
	}
}

func NewRestClient(url, token string) *RestClient {
	return &RestClient{
		token: token,
		url:   url,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewAPIClient(graphClient *GraphQLClient, restClient *RestClient) *APIClient {
	return &APIClient{
		graphClient,
		restClient,
	}
}

//func NewAPIClient(graphClient *GraphQLClient) *APIClient {
//	return &APIClient{
//		graphClient,
//	}
//}

//func NewRestClient(url, token string) *RestClient {
//todo use authorized or whatever
//return &RestClient{Client: http.}
//}

func (c RestClient) CreateHttpSource(ctx context.Context, input client.CreateHttpSourceInput) (*client.HttpSource, error) {
	jsonData, err := json.Marshal(input)
	tflog.Warn(ctx, "req: ", map[string]interface{}{
		"body": string(jsonData),
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling data: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create: %w", err)
	}
	req.Header.Add("X-API-Key", c.token)
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, err := getErrorResponseMsg(resp)
		if err != nil {
			msg = err.Error()
		}
		return nil, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, msg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	tflog.Warn(ctx, "resp: ", map[string]interface{}{
		"body": string(body),
	})

	var response *client.HttpSource
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c RestClient) UpdateHttpSource(ctx context.Context, input client.UpdateHttpSourceInput) (*client.HttpSource, error) {
	reqURL := fmt.Sprintf("%s/%s", c.url, input.Id)
	tflog.Warn(ctx, "req: ", map[string]interface{}{
		"body": input.Id,
	})
	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create: %w", err)
	}
	req.Header.Add("X-API-Key", c.token)
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, err := getErrorResponseMsg(resp)
		if err != nil {
			msg = err.Error()
		}
		return nil, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, msg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response *client.HttpSource
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c RestClient) GetHttpSource(ctx context.Context, id string) (*client.HttpSource, error) {
	reqURL := fmt.Sprintf("%s/%s", c.url, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create: %w", err)
	}
	req.Header.Add("X-API-Key", c.token)
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, err := getErrorResponseMsg(resp)
		if err != nil {
			msg = err.Error()
		}
		return nil, fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, msg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response *client.HttpSource
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

func (c RestClient) DeleteHttpSource(ctx context.Context, id string) error {
	reqURL := fmt.Sprintf("%s/%s", c.url, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create: %w", err)
	}
	req.Header.Add("X-API-Key", c.token)
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		msg, err := getErrorResponseMsg(resp)
		if err != nil {
			msg = err.Error()
		}
		return fmt.Errorf("failed to make request, status: %d, message: %s", resp.StatusCode, msg)
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

func getErrorResponseMsg(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var errResponse *client.HttpErrorResponse
	if err = json.Unmarshal(body, &errResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body to get error response: %w", err)
	}

	return errResponse.Message, nil
}
