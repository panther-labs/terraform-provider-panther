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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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

func TestHandleReadError_Unauthorized(t *testing.T) {
	resp := &resource.ReadResponse{}
	err := &client.APIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"}
	handled := handleReadError(context.Background(), resp, "Test", "id-1", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Authentication failed")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "PANTHER_API_TOKEN")
}

func TestHandleReadError_Forbidden(t *testing.T) {
	resp := &resource.ReadResponse{}
	err := &client.APIError{StatusCode: http.StatusForbidden, Message: "forbidden"}
	handled := handleReadError(context.Background(), resp, "Test", "id-1", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Insufficient permissions")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "permission")
}

func TestHandleCreateError_Unauthorized(t *testing.T) {
	resp := &resource.CreateResponse{}
	err := &client.APIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"}
	handled := handleCreateError(resp, "Test", err)
	assert.True(t, handled)
	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Authentication failed")
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

func TestApplySchemaOverrides_Default(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"field1": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "field1", Default: stringdefault.StaticString("")},
	})
	attr := s.Attributes["field1"].(schema.StringAttribute)
	assert.NotNil(t, attr.Default)
}

func TestApplySchemaOverrides_Sensitive(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"secret": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "secret", Default: stringdefault.StaticString(""), Sensitive: true},
	})
	attr := s.Attributes["secret"].(schema.StringAttribute)
	assert.True(t, attr.Sensitive)
	assert.NotNil(t, attr.Default)
}

func TestApplySchemaOverrides_PlanModifiers(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"derived": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "derived", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
	})
	attr := s.Attributes["derived"].(schema.StringAttribute)
	assert.Len(t, attr.PlanModifiers, 1)
}

func TestApplySchemaOverrides_MissingAttribute(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{},
	}
	// Should not panic
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "nonexistent", Default: stringdefault.StaticString("")},
	})
}

func TestApplySchemaOverrides_NonStringAttribute(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"count": schema.Int64Attribute{Optional: true},
		},
	}
	// Should skip non-string attributes without panic
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "count", Default: stringdefault.StaticString("")},
	})
	// Attribute should be unchanged
	_, ok := s.Attributes["count"].(schema.Int64Attribute)
	assert.True(t, ok)
}

func TestApplySchemaOverrides_MultipleOverrides(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"field_a":     schema.StringAttribute{Optional: true, Computed: true},
			"field_b":     schema.StringAttribute{Optional: true, Computed: true},
			"nonexistent": schema.Int64Attribute{Optional: true}, // wrong type, should be skipped
		},
	}
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "field_a", Default: stringdefault.StaticString(""), Sensitive: true},
		{Name: "field_b", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		{Name: "nonexistent", Default: stringdefault.StaticString("")}, // skipped (Int64, not String)
		{Name: "missing", Default: stringdefault.StaticString("")},     // skipped (doesn't exist)
	})

	attrA := s.Attributes["field_a"].(schema.StringAttribute)
	assert.NotNil(t, attrA.Default)
	assert.True(t, attrA.Sensitive)
	assert.Empty(t, attrA.PlanModifiers) // not set for field_a

	attrB := s.Attributes["field_b"].(schema.StringAttribute)
	assert.Nil(t, attrB.Default)     // not set for field_b
	assert.False(t, attrB.Sensitive) // not set for field_b
	assert.Len(t, attrB.PlanModifiers, 1)
}

// assertNoOptionalComputedWithoutDefault checks that every Optional+Computed string attribute
// in the schema has a Default set. Call this in tests after Schema() to catch missing overrides
// when the generated schema adds new fields.
func assertNoOptionalComputedWithoutDefault(t *testing.T, s schema.Schema) {
	t.Helper()
	for name, raw := range s.Attributes {
		attr, ok := raw.(schema.StringAttribute)
		if !ok {
			continue
		}
		if attr.Optional && attr.Computed && attr.Default == nil && len(attr.PlanModifiers) == 0 {
			t.Errorf("attribute %q is Optional+Computed without a Default or PlanModifier — add it to applySchemaOverrides or set a default in the generated schema", name)
		}
	}
}

func TestHttpsourceSchema_AllOptionalComputedHaveDefaults(t *testing.T) {
	r := &httpsourceResource{}
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), req, resp)
	assertNoOptionalComputedWithoutDefault(t, resp.Schema)
}

func TestPubsubsourceSchema_AllOptionalComputedHaveDefaults(t *testing.T) {
	r := &pubsubsourceResource{}
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), req, resp)
	assertNoOptionalComputedWithoutDefault(t, resp.Schema)
}
