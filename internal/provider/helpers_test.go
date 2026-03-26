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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHandleReadError(t *testing.T) {
	tests := []struct {
		name                string
		err                 error
		initState           bool
		wantHandled         bool
		wantHasError        bool
		wantSummaryContains string
		wantDetailContains  string
	}{
		{"Nil", nil, false, false, false, "", ""},
		{"NotFound",
			&client.APIError{StatusCode: http.StatusNotFound, Message: "not found"},
			true, true, false, "", ""},
		{"OtherError",
			fmt.Errorf("connection refused"),
			false, true, true, "Error reading Test", ""},
		{"Unauthorized",
			&client.APIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"},
			false, true, true, "Authentication failed", "PANTHER_API_TOKEN"},
		{"Forbidden",
			&client.APIError{StatusCode: http.StatusForbidden, Message: "forbidden"},
			false, true, true, "Insufficient permissions", "permission"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &resource.ReadResponse{}
			if tt.initState {
				// RemoveResource needs a schema to avoid panic
				resp.State = tfsdk.State{
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"id": schema.StringAttribute{Computed: true},
						},
					},
				}
			}
			handled := handleReadError(context.Background(), resp, "Test", "id-1", tt.err)
			assert.Equal(t, tt.wantHandled, handled)
			assert.Equal(t, tt.wantHasError, resp.Diagnostics.HasError())
			if tt.wantSummaryContains != "" {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), tt.wantSummaryContains)
			}
			if tt.wantDetailContains != "" {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tt.wantDetailContains)
			}
		})
	}
}

func TestHandleCreateError(t *testing.T) {
	tests := []struct {
		name                string
		err                 error
		wantHandled         bool
		wantHasError        bool
		wantSummaryContains string
		wantDetailContains  string
	}{
		{"Nil", nil, false, false, "", ""},
		{"Unauthorized",
			&client.APIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"},
			true, true, "Authentication failed", ""},
		{"Conflict",
			&client.APIError{StatusCode: http.StatusConflict, Message: "label already exists"},
			true, true, "already exists", "terraform import"},
		{"OtherError",
			fmt.Errorf("connection refused"),
			true, true, "Error creating Test", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &resource.CreateResponse{}
			handled := handleCreateError(resp, "Test", tt.err)
			assert.Equal(t, tt.wantHandled, handled)
			assert.Equal(t, tt.wantHasError, resp.Diagnostics.HasError())
			if tt.wantSummaryContains != "" {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), tt.wantSummaryContains)
			}
			if tt.wantDetailContains != "" {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tt.wantDetailContains)
			}
		})
	}
}

func TestHandleUpdateError(t *testing.T) {
	tests := []struct {
		name                string
		err                 error
		wantHandled         bool
		wantHasError        bool
		wantSummaryContains string
		wantDetailContains  string
	}{
		{"Nil", nil, false, false, "", ""},
		{"Conflict",
			&client.APIError{StatusCode: http.StatusConflict, Message: "label already exists"},
			true, true, "Conflict updating Test", "conflicts with an existing resource"},
		{"OtherError",
			fmt.Errorf("connection refused"),
			true, true, "Error updating Test", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &resource.UpdateResponse{}
			handled := handleUpdateError(resp, "Test", "id-1", tt.err)
			assert.Equal(t, tt.wantHandled, handled)
			assert.Equal(t, tt.wantHasError, resp.Diagnostics.HasError())
			if tt.wantSummaryContains != "" {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), tt.wantSummaryContains)
			}
			if tt.wantDetailContains != "" {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tt.wantDetailContains)
			}
		})
	}
}

func TestHandleDeleteError(t *testing.T) {
	tests := []struct {
		name                string
		err                 error
		wantHandled         bool
		wantHasError        bool
		wantSummaryContains string
	}{
		{"Nil", nil, false, false, ""},
		{"NotFound",
			&client.APIError{StatusCode: http.StatusNotFound, Message: "not found"},
			false, false, ""},
		{"OtherError",
			fmt.Errorf("connection refused"),
			true, true, "Error deleting Test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &resource.DeleteResponse{}
			handled := handleDeleteError(resp, "Test", "id-1", tt.err)
			assert.Equal(t, tt.wantHandled, handled)
			assert.Equal(t, tt.wantHasError, resp.Diagnostics.HasError())
			if tt.wantSummaryContains != "" {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), tt.wantSummaryContains)
			}
		})
	}
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
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "count", Default: stringdefault.StaticString("")},
	})
	_, ok := s.Attributes["count"].(schema.Int64Attribute)
	assert.True(t, ok)
}

func TestApplySchemaOverrides_MultipleOverrides(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"field_a":     schema.StringAttribute{Optional: true, Computed: true},
			"field_b":     schema.StringAttribute{Optional: true, Computed: true},
			"nonexistent": schema.Int64Attribute{Optional: true},
		},
	}
	applySchemaOverrides(&s, []SchemaOverride{
		{Name: "field_a", Default: stringdefault.StaticString(""), Sensitive: true},
		{Name: "field_b", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		{Name: "nonexistent", Default: stringdefault.StaticString("")},
		{Name: "missing", Default: stringdefault.StaticString("")},
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

func TestPatchIDAttribute_Normal(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
		},
	}
	patchIDAttribute(&s)
	attr := s.Attributes["id"].(schema.StringAttribute)
	assert.Len(t, attr.PlanModifiers, 1)
}

func TestPatchIDAttribute_MissingID(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{},
	}
	patchIDAttribute(&s) // should not panic
}

func TestPatchIDAttribute_WrongType(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{Computed: true},
		},
	}
	patchIDAttribute(&s) // should not panic; attribute unchanged
	_, ok := s.Attributes["id"].(schema.Int64Attribute)
	assert.True(t, ok)
}

func TestConvertLogTypes(t *testing.T) {
	ctx := context.Background()
	diags := diag.Diagnostics{}

	list, d := types.ListValueFrom(ctx, types.StringType, []string{"AWS.CloudTrail", "AWS.S3"})
	diags.Append(d...)
	require.False(t, diags.HasError())

	result := convertLogTypes(ctx, list, diags)
	assert.Equal(t, []string{"AWS.CloudTrail", "AWS.S3"}, result)
}

func TestConvertLogTypes_Empty(t *testing.T) {
	ctx := context.Background()
	diags := diag.Diagnostics{}

	list, d := types.ListValueFrom(ctx, types.StringType, []string{})
	diags.Append(d...)

	result := convertLogTypes(ctx, list, diags)
	assert.Empty(t, result)
}

func TestConvertFromLogTypes(t *testing.T) {
	ctx := context.Background()
	diags := diag.Diagnostics{}

	list := convertFromLogTypes(ctx, []string{"AWS.CloudTrail"}, diags)
	assert.False(t, diags.HasError())
	assert.False(t, list.IsNull())
	assert.Equal(t, 1, len(list.Elements()))
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
