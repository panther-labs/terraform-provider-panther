package panther

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

// Saas customer which provides the graphql endpoint (legacy behavior)
func TestCreateAPIClient_SaasGraphEndpoint(t *testing.T) {
	url := "panther-url/public/graphql"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/log-sources/http", client.RestClient.url)
}

// Saas customer which provides the panther root url (new behavior)
func TestCreateAPIClient_SaasBaseURL(t *testing.T) {
	url := "panther-url"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/log-sources/http", client.RestClient.url)
}

// Cloud Connected/Self-hosted customer which provides the graphql endpoint (legacy behavior)
func TestCreateAPIClient_CCGraphEndpoint(t *testing.T) {
	url := "panther-url/v1/public/graphql"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/v1/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/v1/log-sources/http", client.RestClient.url)
}

// Cloud Connected/Self-hosted customer which provides the panther root url (new behavior)
func TestCreateAPIClient_CCBaseURL(t *testing.T) {
	url := "panther-url/v1"
	client := *CreateAPIClient(url, "token")
	graphUrl := reflect.ValueOf(client).FieldByIndex([]int{0}).Elem().FieldByName("url").String()
	assert.Equal(t, "panther-url/v1/public/graphql", graphUrl)
	assert.Equal(t, "panther-url/v1/log-sources/http", client.RestClient.url)
}
