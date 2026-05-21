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
	"context"
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-panther/internal/client"
	"time"

	"github.com/hasura/go-graphql-client"
)

const GraphqlPath = "/public/graphql"

type apiKeyTransport struct {
	token string
	next  http.RoundTripper
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("X-API-Key", t.token)
	return t.next.RoundTrip(r)
}

func NewAuthorizedHTTPClient(token string) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &apiKeyTransport{
			token: token,
			next:  http.DefaultTransport,
		},
	}
}

var _ client.GraphQLClient = (*GraphQLClient)(nil)

type GraphQLClient struct {
	*graphql.Client
}

func NewGraphQLClient(url, token string) *GraphQLClient {
	// Strip the legacy /public/graphql suffix for backwards compatibility.
	pantherUrl := strings.TrimSuffix(url, GraphqlPath)
	return &GraphQLClient{
		graphql.NewClient(
			fmt.Sprintf("%s%s", pantherUrl, GraphqlPath),
			NewAuthorizedHTTPClient(token)),
	}
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


// Cloud Account GraphQL methods
func (c *GraphQLClient) CreateCloudAccount(ctx context.Context, input client.CreateCloudAccountInput) (client.CreateCloudAccountOutput, error) {
	var m struct {
		CreateCloudAccount struct {
			client.CreateCloudAccountOutput
		} `graphql:"createCloudAccount(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("CreateCloudAccount"))
	if err != nil {
		return client.CreateCloudAccountOutput{}, fmt.Errorf("GraphQL mutation failed: %w", err)
	}
	return m.CreateCloudAccount.CreateCloudAccountOutput, nil
}

func (c *GraphQLClient) UpdateCloudAccount(ctx context.Context, input client.UpdateCloudAccountInput) (client.UpdateCloudAccountOutput, error) {
	var m struct {
		UpdateCloudAccount struct {
			client.UpdateCloudAccountOutput
		} `graphql:"updateCloudAccount(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("UpdateCloudAccount"))
	if err != nil {
		return client.UpdateCloudAccountOutput{}, fmt.Errorf("GraphQL mutation failed: %w", err)
	}
	return m.UpdateCloudAccount.UpdateCloudAccountOutput, nil
}

func (c *GraphQLClient) GetCloudAccount(ctx context.Context, id string) (*client.CloudAccount, error) {
	var q struct {
		CloudAccount client.CloudAccount `graphql:"cloudAccount(id: $id)"`
	}

	err := c.Query(ctx, &q, map[string]interface{}{
		"id": graphql.ID(id),
	}, graphql.OperationName("CloudAccount"))
	if err != nil {
		return nil, fmt.Errorf("GraphQL query failed: %w", err)
	}
	return &q.CloudAccount, nil
}

func (c *GraphQLClient) DeleteCloudAccount(ctx context.Context, input client.DeleteCloudAccountInput) (client.DeleteCloudAccountOutput, error) {
	var m struct {
		DeleteCloudAccount struct {
			ID string `graphql:"id"`
		} `graphql:"deleteCloudAccount(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("DeleteCloudAccount"))
	if err != nil {
		return client.DeleteCloudAccountOutput{}, fmt.Errorf("GraphQL mutation failed: %w", err)
	}
	return client.DeleteCloudAccountOutput{ID: m.DeleteCloudAccount.ID}, nil
}

// Schema GraphQL methods
func (c *GraphQLClient) CreateSchema(ctx context.Context, input client.CreateSchemaInput) (client.CreateSchemaOutput, error) {
	var m struct {
		CreateOrUpdateSchema struct {
			client.CreateSchemaOutput
		} `graphql:"createOrUpdateSchema(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("CreateOrUpdateSchema"))
	if err != nil {
		return client.CreateSchemaOutput{}, fmt.Errorf("GraphQL mutation failed: %w", err)
	}
	return m.CreateOrUpdateSchema.CreateSchemaOutput, nil
}

func (c *GraphQLClient) UpdateSchema(ctx context.Context, input client.UpdateSchemaInput) (client.UpdateSchemaOutput, error) {
	var m struct {
		CreateOrUpdateSchema struct {
			client.UpdateSchemaOutput
		} `graphql:"createOrUpdateSchema(input: $input)"`
	}
	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("CreateOrUpdateSchema"))
	if err != nil {
		return client.UpdateSchemaOutput{}, fmt.Errorf("GraphQL mutation failed: %w", err)
	}
	return m.CreateOrUpdateSchema.UpdateSchemaOutput, nil
}

func (c *GraphQLClient) GetSchema(ctx context.Context, name string) (*client.Schema, error) {
	var q struct {
		Schemas struct {
			Edges []struct {
				Node client.Schema `graphql:"node"`
			} `graphql:"edges"`
		} `graphql:"schemas(input: $input)"`
	}

	err := c.Query(ctx, &q, map[string]interface{}{
		"input": map[string]interface{}{
			"cursor": "",
		},
	}, graphql.OperationName("Schemas"))
	if err != nil {
		return nil, fmt.Errorf("GraphQL query failed: %w", err)
	}

	// Find schema by name
	for _, edge := range q.Schemas.Edges {
		if edge.Node.Name == name {
			return &edge.Node, nil
		}
	}

	return nil, nil // Schema not found
}

func (c *GraphQLClient) DeleteSchema(ctx context.Context, input client.DeleteSchemaInput) (client.DeleteSchemaOutput, error) {
	var m struct {
		UpdateSchemaStatus struct {
			Schema struct {
				Name string `graphql:"name"`
			} `graphql:"schema"`
		} `graphql:"updateSchemaStatus(input: $input)"`
	}

	err := c.Mutate(ctx, &m, map[string]interface{}{
		"input": map[string]interface{}{
			"name":       input.Name,
			"isArchived": true,
		},
	}, graphql.OperationName("UpdateSchemaStatus"))
	if err != nil {
		return client.DeleteSchemaOutput{}, fmt.Errorf("GraphQL mutation failed: %w", err)
	}

	return client.DeleteSchemaOutput{Name: m.UpdateSchemaStatus.Schema.Name}, nil
}
