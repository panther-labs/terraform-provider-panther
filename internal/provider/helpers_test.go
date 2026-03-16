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
	"context"
	"fmt"
	"net/http"
	"testing"

	"terraform-provider-panther/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

func TestHandleReadError_Nil(t *testing.T) {
	resp := &resource.ReadResponse{}
	handled := handleReadError(context.Background(), resp, "Test", "id-1", nil)
	assert.False(t, handled)
	assert.False(t, resp.Diagnostics.HasError())
}

func TestHandleReadError_NotFound(t *testing.T) {
	resp := &resource.ReadResponse{}
	// Initialize state with a minimal schema so RemoveResource doesn't panic
	resp.State = tfsdk.State{
		Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{Computed: true},
			},
		},
	}
	err := &client.APIError{StatusCode: http.StatusNotFound, Message: "not found"}
	handled := handleReadError(context.Background(), resp, "Test", "id-1", err)
	assert.True(t, handled)
	assert.False(t, resp.Diagnostics.HasError()) // 404 removes from state, not an error diagnostic
}

func TestHandleReadError_OtherError(t *testing.T) {
	resp := &resource.ReadResponse{}
	err := fmt.Errorf("connection refused")
	handled := handleReadError(context.Background(), resp, "Test", "id-1", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Error reading Test")
}

func TestHandleCreateError_Nil(t *testing.T) {
	resp := &resource.CreateResponse{}
	handled := handleCreateError(resp, "Test", nil)
	assert.False(t, handled)
	assert.False(t, resp.Diagnostics.HasError())
}

func TestHandleCreateError_Conflict(t *testing.T) {
	resp := &resource.CreateResponse{}
	err := &client.APIError{StatusCode: http.StatusConflict, Message: "label already exists"}
	handled := handleCreateError(resp, "Test", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "already exists")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "terraform import")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "already exists")
}

func TestHandleCreateError_OtherError(t *testing.T) {
	resp := &resource.CreateResponse{}
	err := fmt.Errorf("connection refused")
	handled := handleCreateError(resp, "Test", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Error creating Test")
}

func TestHandleUpdateError_Nil(t *testing.T) {
	resp := &resource.UpdateResponse{}
	handled := handleUpdateError(resp, "Test", "id-1", nil)
	assert.False(t, handled)
	assert.False(t, resp.Diagnostics.HasError())
}

func TestHandleUpdateError_Conflict(t *testing.T) {
	resp := &resource.UpdateResponse{}
	err := &client.APIError{StatusCode: http.StatusConflict, Message: "label already exists"}
	handled := handleUpdateError(resp, "Test", "id-1", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Conflict updating Test")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "conflicts with an existing resource")
}

func TestHandleUpdateError_OtherError(t *testing.T) {
	resp := &resource.UpdateResponse{}
	err := fmt.Errorf("connection refused")
	handled := handleUpdateError(resp, "Test", "id-1", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Error updating Test")
}

func TestHandleDeleteError_Nil(t *testing.T) {
	resp := &resource.DeleteResponse{}
	handled := handleDeleteError(resp, "Test", "id-1", nil)
	assert.False(t, handled)
}

func TestHandleDeleteError_NotFound(t *testing.T) {
	resp := &resource.DeleteResponse{}
	err := &client.APIError{StatusCode: http.StatusNotFound, Message: "not found"}
	handled := handleDeleteError(resp, "Test", "id-1", err)
	assert.False(t, handled) // 404 on delete = success, not handled as error
	assert.False(t, resp.Diagnostics.HasError())
}

func TestHandleDeleteError_OtherError(t *testing.T) {
	resp := &resource.DeleteResponse{}
	err := fmt.Errorf("connection refused")
	handled := handleDeleteError(resp, "Test", "id-1", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Error deleting Test")
}
