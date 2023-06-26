package panther

import (
	"context"

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

func (c *Client) CreateS3Source(ctx context.Context, source client.CreateS3SourceInput) (client.CreateS3SourceOutput, error) {
	var m struct {
		CreateS3Source struct {
			client.CreateS3SourceOutput
		} `graphql:"createS3Source(input: $input)"`
	}
	err := c.WithDebug(true).Mutate(ctx, &m, map[string]interface{}{
		"input": source,
	}, graphql.OperationName("CreateS3Source"))
	if err != nil {
		return client.CreateS3SourceOutput{}, err
	}
	return m.CreateS3Source.CreateS3SourceOutput, nil
}
