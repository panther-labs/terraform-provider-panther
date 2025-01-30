package panther

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

// Customer with custom url that provides the graphql endpoint (legacy behavior)
func TestCreateAPIClient_CustomURLWithGraphEndpoint(t *testing.T) {
	url := "panther-url/public/graphql"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/log-sources/http", client.RestClient.url)
}

// Customer with custom url that provides the panther root url (new behavior)
func TestCreateAPIClient_CustomUrlWithBaseUrl(t *testing.T) {
	url := "panther-url"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/log-sources/http", client.RestClient.url)
}

// Customer with API Gateway url that provides the graphql endpoint (legacy behavior)
func TestCreateAPIClient_ApiGWUrlWithGraphEndpoint(t *testing.T) {
	url := "panther-url/v1/public/graphql"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/v1/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/v1/log-sources/http", client.RestClient.url)
}

// Customer with API Gateway url that provides the panther root url (new behavior)
func TestCreateAPIClient_ApiGWUrlWithBaseUrl(t *testing.T) {
	url := "panther-url/v1"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/v1/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/v1/log-sources/http", client.RestClient.url)
}
