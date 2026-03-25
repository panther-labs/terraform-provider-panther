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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTransport struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func TestAuthTransport_SetsAPIKeyHeader(t *testing.T) {
	var captured *http.Request
	transport := &authTransport{
		token: "test-token",
		next: &mockTransport{handler: func(req *http.Request) (*http.Response, error) {
			captured = req
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		}},
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/things", nil)
	_, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, "test-token", captured.Header.Get("X-API-Key"))
}

func TestAuthTransport_SetsContentTypeOnlyWithBody(t *testing.T) {
	var captured *http.Request
	transport := &authTransport{
		token: "test-token",
		next: &mockTransport{handler: func(req *http.Request) (*http.Response, error) {
			captured = req
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		}},
	}

	// GET without body — no Content-Type
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/things", nil)
	_, _ = transport.RoundTrip(req)
	assert.Empty(t, captured.Header.Get("Content-Type"))

	// POST with body — Content-Type set
	req, _ = http.NewRequest(http.MethodPost, "https://api.example.com/things", strings.NewReader(`{}`))
	_, _ = transport.RoundTrip(req)
	assert.Equal(t, "application/json", captured.Header.Get("Content-Type"))
}

func TestAuthTransport_ClonesRequest(t *testing.T) {
	transport := &authTransport{
		token: "test-token",
		next: &mockTransport{handler: func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		}},
	}

	original, _ := http.NewRequest(http.MethodGet, "https://api.example.com/things", nil)
	_, _ = transport.RoundTrip(original)

	// Original request should NOT have the API key header (clone was modified, not original)
	assert.Empty(t, original.Header.Get("X-API-Key"))
}

func TestAuthTransport_PassesThroughErrors(t *testing.T) {
	transport := &authTransport{
		token: "test-token",
		next: &mockTransport{handler: func(req *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		}},
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/things", nil)
	_, err := transport.RoundTrip(req)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestNewHTTPClient_Timeout(t *testing.T) {
	client := NewHTTPClient("token")
	assert.Equal(t, 10*1e9, float64(client.Timeout)) // 10 seconds in nanoseconds
}
