package panther

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// graphqlURL extracts the unexported url field from the underlying graphql.Client
// via reflection, so we can verify that URL trimming logic works correctly.
func graphqlURL(c ProviderClients) string {
	return reflect.ValueOf(c.GraphQL).Elem().FieldByName("url").String()
}

// Customer with custom url that provides the graphql endpoint (legacy behavior)
func TestNewProviderClients_CustomURLWithGraphEndpoint(t *testing.T) {
	c := *NewProviderClients("panther-url/public/graphql", "token")
	assert.Equal(t, "panther-url/public/graphql", graphqlURL(c))
	assert.Equal(t, "panther-url", c.REST.BaseURL)
}

// Customer with custom url that provides the panther root url (new behavior)
func TestNewProviderClients_CustomUrlWithBaseUrl(t *testing.T) {
	c := *NewProviderClients("panther-url", "token")
	assert.Equal(t, "panther-url/public/graphql", graphqlURL(c))
	assert.Equal(t, "panther-url", c.REST.BaseURL)
}

// Customer with API Gateway url that provides the graphql endpoint (legacy behavior)
func TestNewProviderClients_ApiGWUrlWithGraphEndpoint(t *testing.T) {
	c := *NewProviderClients("panther-url/v1/public/graphql", "token")
	assert.Equal(t, "panther-url/v1/public/graphql", graphqlURL(c))
	assert.Equal(t, "panther-url/v1", c.REST.BaseURL)
}

// Customer with API Gateway url that provides the panther root url (new behavior)
func TestNewProviderClients_ApiGWUrlWithBaseUrl(t *testing.T) {
	c := *NewProviderClients("panther-url/v1", "token")
	assert.Equal(t, "panther-url/v1/public/graphql", graphqlURL(c))
	assert.Equal(t, "panther-url/v1", c.REST.BaseURL)
}
