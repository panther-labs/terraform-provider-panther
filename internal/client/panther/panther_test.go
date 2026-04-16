package panther

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests verify that NewRESTClient correctly strips /public/graphql from the URL
// (backwards compatibility for users who configured the old URL format)
// and sets BaseURL to the root.

func TestNewRESTClient_CustomURLWithGraphEndpoint(t *testing.T) {
	c := NewRESTClient("panther-url/public/graphql", "token")
	assert.Equal(t, "panther-url", c.BaseURL)
}

func TestNewRESTClient_CustomUrlWithBaseUrl(t *testing.T) {
	c := NewRESTClient("panther-url", "token")
	assert.Equal(t, "panther-url", c.BaseURL)
}

func TestNewRESTClient_ApiGWUrlWithGraphEndpoint(t *testing.T) {
	c := NewRESTClient("panther-url/v1/public/graphql", "token")
	assert.Equal(t, "panther-url/v1", c.BaseURL)
}

func TestNewRESTClient_ApiGWUrlWithBaseUrl(t *testing.T) {
	c := NewRESTClient("panther-url/v1", "token")
	assert.Equal(t, "panther-url/v1", c.BaseURL)
}
