package panther

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests verify that NewProviderClients correctly strips /public/graphql from the URL
// (backwards compatibility) and sets REST.BaseURL to the root. The GraphQL URL is
// always BaseURL + GraphQLPath, so testing BaseURL is sufficient.

func TestNewProviderClients_CustomURLWithGraphEndpoint(t *testing.T) {
	c := NewProviderClients("panther-url/public/graphql", "token")
	assert.Equal(t, "panther-url", c.REST.BaseURL)
}

func TestNewProviderClients_CustomUrlWithBaseUrl(t *testing.T) {
	c := NewProviderClients("panther-url", "token")
	assert.Equal(t, "panther-url", c.REST.BaseURL)
}

func TestNewProviderClients_ApiGWUrlWithGraphEndpoint(t *testing.T) {
	c := NewProviderClients("panther-url/v1/public/graphql", "token")
	assert.Equal(t, "panther-url/v1", c.REST.BaseURL)
}

func TestNewProviderClients_ApiGWUrlWithBaseUrl(t *testing.T) {
	c := NewProviderClients("panther-url/v1", "token")
	assert.Equal(t, "panther-url/v1", c.REST.BaseURL)
}
