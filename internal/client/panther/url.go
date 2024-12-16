package panther

import "strings"

const GraphqlEndpoint = "/v1/public/graphql"
const RestEndpoint = "/v1/log-sources/http"

func TrimUrl(url string) string {
	if strings.HasSuffix(url, GraphqlEndpoint) {
		return strings.TrimSuffix(url, GraphqlEndpoint)
	}
	return url
}
