package panther

import (
	"net/http"
	"time"
)

type AuthorizedHTTPClient struct {
	http.Client
	token string
}

func NewAuthorizedHTTPClient(token string) *AuthorizedHTTPClient {
	return &AuthorizedHTTPClient{
		Client: http.Client{
			Timeout: 3 * time.Second,
		},
		token: token,
	}
}

func (c *AuthorizedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-API-Key", c.token)
	return c.Client.Do(req)
}
