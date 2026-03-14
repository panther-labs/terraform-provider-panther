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

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestProviderClients_NilProviderData(t *testing.T) {
	req := resource.ConfigureRequest{ProviderData: nil}
	resp := &resource.ConfigureResponse{}

	c := providerClients(req, resp)

	assert.Nil(t, c)
	assert.False(t, resp.Diagnostics.HasError())
}

func TestProviderClients_WrongType(t *testing.T) {
	req := resource.ConfigureRequest{ProviderData: "wrong-type"}
	resp := &resource.ConfigureResponse{}

	c := providerClients(req, resp)

	assert.Nil(t, c)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Unexpected Resource Configure Type")
}
