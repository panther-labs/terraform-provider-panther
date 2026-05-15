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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	client := newHTTPClient("token", "ua")
	assert.Equal(t, 30*time.Second, client.Timeout)
}

func TestBuildUserAgent_FormatAndFallbacks(t *testing.T) {
	tests := []struct {
		name             string
		providerVersion  string
		terraformVersion string
		want             string
	}{
		{
			name:             "Populated",
			providerVersion:  "0.5.2",
			terraformVersion: "1.10.4",
			want:             "Terraform/1.10.4 terraform-provider-panther/0.5.2",
		},
		{
			name:             "EmptyProviderVersionFallsBackToDev",
			providerVersion:  "",
			terraformVersion: "1.10.4",
			want:             "Terraform/1.10.4 terraform-provider-panther/dev",
		},
		{
			name:             "EmptyTerraformVersionFallsBackToUnknown",
			providerVersion:  "0.5.2",
			terraformVersion: "",
			want:             "Terraform/unknown terraform-provider-panther/0.5.2",
		},
		{
			name:             "DevLiteral",
			providerVersion:  "dev",
			terraformVersion: "1.10.4",
			want:             "Terraform/1.10.4 terraform-provider-panther/dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildUserAgent(tt.providerVersion, tt.terraformVersion))
		})
	}
}

func TestAuthTransport_SetsUserAgentOnWire(t *testing.T) {
	const wantUA = "Terraform/1.10.4 terraform-provider-panther/0.5.2"

	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newHTTPClient("token", wantUA)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := c.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.Equal(t, wantUA, gotUA)
}
