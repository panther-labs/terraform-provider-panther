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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const graphQLPath = "/public/graphql"

// NewRESTClient creates a configured REST client for the Panther API.
func NewRESTClient(url, token string) *RESTClient {
	// Strip the legacy /public/graphql suffix — older provider configs included it.
	pantherURL := strings.TrimSuffix(url, graphQLPath)
	httpClient := newHTTPClient(token)

	return &RESTClient{
		Doer:    httpClient,
		BaseURL: pantherURL,
	}
}

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type APIError struct {
	StatusCode int
	Message    string
	Method     string
	URL        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s returned status %d: %s", e.Method, e.URL, e.StatusCode, e.Message)
}

func hasStatusCode(err error, code int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == code
}

// IsNotFound reports whether err is an HTTP 404.
func IsNotFound(err error) bool { return hasStatusCode(err, http.StatusNotFound) }

// IsConflict reports whether err is an HTTP 409 (e.g. duplicate integration label).
func IsConflict(err error) bool { return hasStatusCode(err, http.StatusConflict) }

// IsUnauthorized reports whether err is an HTTP 401 (bad or expired token).
func IsUnauthorized(err error) bool { return hasStatusCode(err, http.StatusUnauthorized) }

// IsForbidden reports whether err is an HTTP 403 (insufficient permissions).
func IsForbidden(err error) bool { return hasStatusCode(err, http.StatusForbidden) }

type RESTClient struct {
	Doer    Doer
	BaseURL string
}

func isHTTPSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// restExec builds the URL, creates the request, executes it, and checks for 2xx.
// On success it returns the open *http.Response (caller must close Body).
// On failure it returns an *APIError.
func restExec(ctx context.Context, c *RESTClient, method, path string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	if !isHTTPSuccess(resp.StatusCode) {
		defer resp.Body.Close()
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    getErrorResponseMsg(resp),
			Method:     method,
			URL:        url,
		}
	}
	return resp, nil
}

// RestDo sends an HTTP request to c.BaseURL+path, marshals body as JSON,
// checks for 2xx success, and unmarshals the response into Resp.
func RestDo[Resp any](ctx context.Context, c *RESTClient, method, path string, body any) (Resp, error) {
	var zero Resp
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}
	resp, err := restExec(ctx, c, method, path, reqBody)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	// Single-resource responses are typically < 10 KB. If list endpoints are added,
	// consider using io.LimitReader here (the error path already limits to 1 MB).
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("failed to read response body (status %d): %w", resp.StatusCode, err)
	}
	var response Resp
	if len(respBody) == 0 {
		return response, nil
	}
	if err = json.Unmarshal(respBody, &response); err != nil {
		return zero, fmt.Errorf("failed to unmarshal response body (status %d): %w", resp.StatusCode, err)
	}
	return response, nil
}

// RestDelete sends a DELETE request to c.BaseURL+path. No response body is read.
func RestDelete(ctx context.Context, c *RESTClient, path string) error {
	resp, err := restExec(ctx, c, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

type httpErrorResponse struct {
	Message string `json:"message"`
}

func getErrorResponseMsg(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	if err != nil {
		return fmt.Sprintf("failed to read error response body: %v", err)
	}

	if len(body) == 0 {
		return "(empty response body)"
	}

	var errResponse httpErrorResponse
	if err = json.Unmarshal(body, &errResponse); err != nil || errResponse.Message == "" {
		// Non-JSON response (e.g. HTML from a load balancer) — return raw body truncated
		const maxDisplay = 512
		if len(body) > maxDisplay {
			return string(body[:maxDisplay]) + "... (truncated)"
		}
		return string(body)
	}

	return errResponse.Message
}
