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
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test helpers ---

type createInput struct {
	Name string `json:"name"`
}

type updateInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type testResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type mockDoer struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func jsonResponse(status int, body any) *http.Response {
	data, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(data)),
	}
}

func newTestResource(doer Doer) *RESTResource[createInput, updateInput, testResp] {
	return NewRESTResource[createInput, updateInput, testResp](
		&RESTClient{Doer: doer, BaseURL: "https://api.example.com"},
		"/things",
		func(u updateInput) string { return u.ID },
	)
}

// --- tests ---

func TestRESTResource_Create(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://api.example.com/things", req.URL.String())

		var input createInput
		require.NoError(t, json.NewDecoder(req.Body).Decode(&input))
		assert.Equal(t, "test-thing", input.Name)

		return jsonResponse(http.StatusCreated, testResp{ID: "id-1", Name: "test-thing"}), nil
	}}

	r := newTestResource(doer)
	resp, err := r.Create(context.Background(), createInput{Name: "test-thing"})

	require.NoError(t, err)
	assert.Equal(t, "id-1", resp.ID)
	assert.Equal(t, "test-thing", resp.Name)
}

func TestRESTResource_Get(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://api.example.com/things/id-1", req.URL.String())
		assert.Nil(t, req.Body)

		return jsonResponse(http.StatusOK, testResp{ID: "id-1", Name: "test-thing"}), nil
	}}

	r := newTestResource(doer)
	resp, err := r.Get(context.Background(), "id-1")

	require.NoError(t, err)
	assert.Equal(t, "id-1", resp.ID)
	assert.Equal(t, "test-thing", resp.Name)
}

func TestRESTResource_Update(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPut, req.Method)
		assert.Equal(t, "https://api.example.com/things/id-1", req.URL.String())

		var input updateInput
		require.NoError(t, json.NewDecoder(req.Body).Decode(&input))
		assert.Equal(t, "updated-name", input.Name)

		return jsonResponse(http.StatusOK, testResp{ID: "id-1", Name: "updated-name"}), nil
	}}

	r := newTestResource(doer)
	resp, err := r.Update(context.Background(), updateInput{ID: "id-1", Name: "updated-name"})

	require.NoError(t, err)
	assert.Equal(t, "id-1", resp.ID)
	assert.Equal(t, "updated-name", resp.Name)
}

func TestRESTResource_Delete(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodDelete, req.Method)
		assert.Equal(t, "https://api.example.com/things/id-1", req.URL.String())

		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	}}

	r := newTestResource(doer)
	err := r.Delete(context.Background(), "id-1")

	require.NoError(t, err)
}

func TestRESTResource_Create_WrongStatus(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, httpErrorResponse{Message: "bad input"}), nil
	}}

	r := newTestResource(doer)
	_, err := r.Create(context.Background(), createInput{Name: "x"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
	assert.Contains(t, err.Error(), "bad input")

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestRESTResource_Delete_WrongStatus(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, httpErrorResponse{Message: "not found"}), nil
	}}

	r := newTestResource(doer)
	err := r.Delete(context.Background(), "id-missing")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
	assert.Contains(t, err.Error(), "not found")
	assert.True(t, IsNotFound(err))
}

func TestRESTResource_NetworkError(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return nil, assert.AnError
	}}

	r := newTestResource(doer)
	_, err := r.Create(context.Background(), createInput{Name: "x"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make request")
}

func TestRESTResource_URL(t *testing.T) {
	noop := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusCreated, testResp{}), nil
	}}

	tests := []struct {
		name     string
		baseURL  string
		path     string
		segments []string
		want     string
	}{
		{"base path only", "https://api.example.com", "/things", nil, "https://api.example.com/things"},
		{"with id segment", "https://api.example.com", "/things", []string{"id-1"}, "https://api.example.com/things/id-1"},
		{"trailing slash on base", "https://api.example.com/", "/things", []string{"id-1"}, "https://api.example.com//things/id-1"},
		{"base with path prefix", "https://api.example.com/v1", "/things", []string{"id-1"}, "https://api.example.com/v1/things/id-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRESTResource[createInput, updateInput, testResp](
				&RESTClient{Doer: noop, BaseURL: tt.baseURL},
				tt.path,
				func(u updateInput) string { return u.ID },
			)
			got := r.url(tt.segments...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Method:     "GET",
		URL:        "https://api.example.com/things/id-1",
		Message:    "not found",
	}
	assert.Equal(t, "GET https://api.example.com/things/id-1 returned status 404: not found", err.Error())
}

func TestAPIErrorPredicates(t *testing.T) {
	tests := []struct {
		name      string
		predicate func(error) bool
		match     *APIError
		noMatch   *APIError
	}{
		{"IsNotFound", IsNotFound, &APIError{StatusCode: http.StatusNotFound}, &APIError{StatusCode: http.StatusConflict}},
		{"IsConflict", IsConflict, &APIError{StatusCode: http.StatusConflict}, &APIError{StatusCode: http.StatusNotFound}},
		{"IsUnauthorized", IsUnauthorized, &APIError{StatusCode: http.StatusUnauthorized}, &APIError{StatusCode: http.StatusForbidden}},
		{"IsForbidden", IsForbidden, &APIError{StatusCode: http.StatusForbidden}, &APIError{StatusCode: http.StatusUnauthorized}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.predicate(tt.match))
			assert.False(t, tt.predicate(tt.noMatch))
			assert.False(t, tt.predicate(assert.AnError))
			assert.False(t, tt.predicate(nil))

			wrapped := fmt.Errorf("context: %w", tt.match)
			assert.True(t, tt.predicate(wrapped))
		})
	}
}

func TestRESTResource_Create_Conflict(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusConflict, httpErrorResponse{Message: "label already exists"}), nil
	}}

	r := newTestResource(doer)
	_, err := r.Create(context.Background(), createInput{Name: "dup"})

	require.Error(t, err)
	assert.True(t, IsConflict(err))
	assert.Contains(t, err.Error(), "label already exists")
}

func TestGetErrorResponseMsg(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantExact    string // exact match; empty means use wantContains
		wantContains string
	}{
		{"ValidJSON", `{"message": "resource not found"}`, "resource not found", ""},
		{"EmptyJSONMessage", `{"message": ""}`, `{"message": ""}`, ""},
		{"MalformedJSON", "not json", "not json", ""},
		{"EmptyBody", "", "(empty response body)", ""},
		{"HTMLBody", "<html><body><h1>502 Bad Gateway</h1></body></html>", "", "502 Bad Gateway"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Body: io.NopCloser(bytes.NewReader([]byte(tt.body))),
			}
			msg := getErrorResponseMsg(resp)
			if tt.wantExact != "" {
				assert.Equal(t, tt.wantExact, msg)
			} else {
				assert.Contains(t, msg, tt.wantContains)
			}
		})
	}
}

func TestGetErrorResponseMsg_LargeBody(t *testing.T) {
	largeBody := bytes.Repeat([]byte("x"), 2<<20) // 2 MB
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewReader(largeBody)),
	}
	msg := getErrorResponseMsg(resp)
	assert.LessOrEqual(t, len(msg), 600)
	assert.Contains(t, msg, "... (truncated)")
}

func TestRestDo_UnmarshalError(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("not valid json"))),
		}, nil
	}}
	r := newTestResource(doer)
	_, err := r.Get(context.Background(), "id-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal response body")
}

func TestRestDo_TransportError(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("connection refused")
	}}
	r := newTestResource(doer)
	_, err := r.Get(context.Background(), "id-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make request")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestRestDo_CancelledContext(t *testing.T) {
	doer := &mockDoer{handler: func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	}}
	r := newTestResource(doer)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.Get(ctx, "id-1")
	require.Error(t, err)
}
