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

// RESTClient holds the shared HTTP transport and base URL for REST API calls.
type RESTClient struct {
	Doer    Doer
	BaseURL string
}

// RESTResource provides typed CRUD operations for a single REST API resource.
// Type parameters:
//
//	CreateIn — the input type for Create (e.g. CreatePubSubSourceInput)
//	UpdateIn — the input type for Update (e.g. UpdatePubSubSourceInput)
//	Resp     — the response type from Create/Get/Update (e.g. PubSubSource)
//
// Conventions assumed: POST (201) for Create, GET (200) for Read, PUT (200) for
// Update, DELETE (204) for Delete, with a single path-segment ID. If a future
// resource diverges (e.g. PATCH updates, 202 async creates, composite URLs),
// extend via functional options on NewRESTResource or call restDo/restDelete
// directly — those helpers have no opinion on method, URL shape, or status code.
type RESTResource[CreateIn, UpdateIn, Resp any] struct {
	client   *RESTClient
	path     string // relative path, e.g. "/log-sources/pubsub"
	updateID func(UpdateIn) string
}

// NewRESTResource constructs a RESTResource for a specific API endpoint.
// The path is relative to the RESTClient's BaseURL.
func NewRESTResource[CreateIn, UpdateIn, Resp any](
	rc *RESTClient,
	path string,
	updateID func(UpdateIn) string,
) *RESTResource[CreateIn, UpdateIn, Resp] {
	return &RESTResource[CreateIn, UpdateIn, Resp]{
		client:   rc,
		path:     path,
		updateID: updateID,
	}
}

func (r *RESTResource[C, U, Resp]) url(segments ...string) string {
	u := r.client.BaseURL + r.path
	for _, s := range segments {
		u += "/" + s
	}
	return u
}

func (r *RESTResource[C, U, Resp]) Create(ctx context.Context, input C) (Resp, error) {
	return restDo[Resp](ctx, r.client.Doer, http.MethodPost, r.url(), http.StatusCreated, input)
}

func (r *RESTResource[C, U, Resp]) Get(ctx context.Context, id string) (Resp, error) {
	return restDo[Resp](ctx, r.client.Doer, http.MethodGet, r.url(id), http.StatusOK, nil)
}

func (r *RESTResource[C, U, Resp]) Update(ctx context.Context, input U) (Resp, error) {
	return restDo[Resp](ctx, r.client.Doer, http.MethodPut, r.url(r.updateID(input)), http.StatusOK, input)
}

func (r *RESTResource[C, U, Resp]) Delete(ctx context.Context, id string) error {
	return restDelete(ctx, r.client.Doer, r.url(id))
}

// restDo is a generic helper that handles the common REST pattern:
// marshal request body → send HTTP request → check status → unmarshal typed response.
func restDo[Resp any](ctx context.Context, doer Doer, method, url string, expectedStatus int, body any) (Resp, error) {
	var zero Resp
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
	resp, err := doer.Do(req)
	if err != nil {
		return zero, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return zero, &APIError{
			StatusCode: resp.StatusCode,
			Message:    getErrorResponseMsg(resp),
			Method:     method,
			URL:        url,
		}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("failed to read response body: %w", err)
	}

	var response Resp
	if err = json.Unmarshal(respBody, &response); err != nil {
		return zero, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

// restDelete is a helper for DELETE requests that return no response body.
func restDelete(ctx context.Context, doer Doer, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := doer.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
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
		raw := string(body)
		if len(raw) > maxDisplay {
			raw = raw[:maxDisplay] + "... (truncated)"
		}
		return raw
	}

	return errResponse.Message
}
