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
)

// Doer abstracts HTTP request execution for testability.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// APIError represents an HTTP API error with a status code for programmatic handling.
type APIError struct {
	StatusCode int
	Message    string
	Method     string
	URL        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s returned status %d: %s", e.Method, e.URL, e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is an HTTP 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// IsConflict returns true if the error is an HTTP 409 (e.g. duplicate integration label).
func IsConflict(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict
}

// IsUnauthorized returns true if the error is an HTTP 401 (bad or expired API token).
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if the error is an HTTP 403 (valid token, insufficient permissions).
func IsForbidden(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden
}

// RESTClient holds the shared HTTP transport and base URL for REST API calls.
type RESTClient struct {
	Doer    Doer
	BaseURL string
}

func isHTTPSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// RestDo sends an HTTP request to c.BaseURL+path, marshals body as JSON,
// checks for 2xx success, and unmarshals the response into Resp.
func RestDo[Resp any](ctx context.Context, c *RESTClient, method, path string, body any) (Resp, error) {
	var zero Resp
	url := c.BaseURL + path
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("error marshaling data: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return zero, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Doer.Do(req)
	if err != nil {
		return zero, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if !isHTTPSuccess(resp.StatusCode) {
		return zero, &APIError{
			StatusCode: resp.StatusCode,
			Message:    getErrorResponseMsg(resp),
			Method:     method,
			URL:        url,
		}
	}

	// Single-resource responses are typically < 10 KB. If list endpoints are added,
	// consider using io.LimitReader here (the error path already limits to 1 MB).
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("failed to read response body (status %d): %w", resp.StatusCode, err)
	}

	var response Resp
	if err = json.Unmarshal(respBody, &response); err != nil {
		return zero, fmt.Errorf("failed to unmarshal response body (status %d): %w", resp.StatusCode, err)
	}

	return response, nil
}

// RestDelete sends a DELETE request to c.BaseURL+path. No response body is expected.
func RestDelete(ctx context.Context, c *RESTClient, path string) error {
	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := c.Doer.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if !isHTTPSuccess(resp.StatusCode) {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    getErrorResponseMsg(resp),
			Method:     http.MethodDelete,
			URL:        url,
		}
	}

	return nil
}

// httpErrorResponse represents an error returned by the Panther REST API.
type httpErrorResponse struct {
	Message string `json:"message"`
}

func getErrorResponseMsg(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	if err != nil {
		return fmt.Sprintf("failed to read response body: %s", err.Error())
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
