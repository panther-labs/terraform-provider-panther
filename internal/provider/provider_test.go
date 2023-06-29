// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"terraform-provider-panther/internal/client"
	"terraform-provider-panther/internal/client/clientfakes"
)

const (
	providerConfig = `
	provider "panther" {
		url = "test.com"
		token = "test-token"
	}
`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
func newTestAccProtoV6ProviderFactories(mockClient clientfakes.FakeClient) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"panther": providerserver.NewProtocol6WithError(NewMock(&mockClient)()),
	}
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func NewMock(mockClient client.Client) func() provider.Provider {
	return func() provider.Provider {
		return &PantherProvider{
			version:        "test",
			clientOverride: mockClient,
		}
	}
}
