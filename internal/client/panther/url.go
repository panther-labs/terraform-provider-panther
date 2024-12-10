package panther

import "strings"

const graphqlEndpoint = "/v1/public/graphql"
const restEndpoint = "/v1/log-sources/http"

func TrimUrl(url string) string {
	if strings.HasSuffix(url, graphqlEndpoint) {
		return strings.TrimSuffix(url, graphqlEndpoint)
	}
	return url
}
