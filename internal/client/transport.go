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

package client

import (
	"net/http"
	"time"
)

type authTransport struct {
	token string
	next  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("X-API-Key", t.token)
	if r.Body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	return t.next.RoundTrip(r)
}

// newHTTPClient returns an *http.Client that satisfies Doer,
// injecting the API key and Content-Type headers on every request.
func newHTTPClient(token string) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &authTransport{
			token: token,
			next:  http.DefaultTransport,
		},
	}
}
