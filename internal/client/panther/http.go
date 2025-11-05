/*
Copyright 2023 Panther Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package panther

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type AuthorizedHTTPClient struct {
	http.Client
	token string
}

func NewAuthorizedHTTPClient(token string) *AuthorizedHTTPClient {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.RetryWaitMin = 500 * time.Millisecond
	rc.RetryWaitMax = 2 * time.Second
	// Quiet default logger to avoid noisy output in Terraform runs
	rc.Logger = nil

	std := rc.StandardClient()
	std.Timeout = 10 * time.Second

	return &AuthorizedHTTPClient{
		Client: *std,
		token:  token,
	}
}

func (c *AuthorizedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-API-Key", c.token)
	return c.Client.Do(req)
}
