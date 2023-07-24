package panther

import (
	"context"
	"fmt"

	"github.com/hasura/go-graphql-client"
	"terraform-provider-panther/internal/client"
)

var _ client.Client = (*Client)(nil)

type Client struct {
	*graphql.Client
}

func NewClient(url, token string) *Client {
	return &Client{
		graphql.NewClient(
			url,
			NewAuthorizedHTTPClient(token)),
	}
}

func (c *Client) CreateS3Source(ctx context.Context, input client.CreateS3SourceInput) (client.CreateS3SourceOutput, error) {
	var m struct {
		CreateS3Source struct {
			client.CreateS3SourceOutput
		} `graphql:"createS3Source(input: $input)"`
	}
	err := c.WithDebug(true).Mutate(ctx, &m, map[string]interface{}{
		"input": input,
	}, graphql.OperationName("CreateS3Source"))
	if err != nil {
		return client.CreateS3SourceOutput{}, fmt.Errorf("GraphQL mutation failed: %v", err)
	}
	return m.CreateS3Source.CreateS3SourceOutput, nil
}

func (c *Client) UpdateS3Source(ctx context.Context, input client.UpdateS3SourceInput) (client.UpdateS3SourceOutput, error) {
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

func (c *Client) DeleteSource(ctx context.Context, input client.DeleteSourceInput) (client.DeleteSourceOutput, error) {
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

func (c *Client) GetS3Source(ctx context.Context, id string) (*client.S3LogIntegration, error) {
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
