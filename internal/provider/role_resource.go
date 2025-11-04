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
	"terraform-provider-panther/internal/client"
	"terraform-provider-panther/internal/client/panther"
	"terraform-provider-panther/internal/provider/resource_role"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*roleResource)(nil)
	_ resource.ResourceWithConfigure   = (*roleResource)(nil)
	_ resource.ResourceWithImportState = (*roleResource)(nil)
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client client.RestClient
}

func (r *roleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_role.RoleResourceSchema(ctx)
	// Add UseStateForUnknown plan modifier to the id attribute
	idAttr := resp.Schema.Attributes["id"].(schema.StringAttribute)
	idAttr.PlanModifiers = append(idAttr.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	resp.Schema.Attributes["id"] = idAttr
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*panther.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *panther.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c.RestClient
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert permissions list to string slice
	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateRoleInput{
		RoleModifiableAttributes: client.RoleModifiableAttributes{
			Name:        data.Name.ValueString(),
			Permissions: permissions,
		},
	}

	// Add optional log type access fields if provided
	if !data.LogTypeAccessKind.IsNull() && !data.LogTypeAccessKind.IsUnknown() {
		input.LogTypeAccessKind = data.LogTypeAccessKind.ValueString()
	}
	if !data.LogTypeAccess.IsNull() && !data.LogTypeAccess.IsUnknown() {
		var logTypeAccess []string
		resp.Diagnostics.Append(data.LogTypeAccess.ElementsAs(ctx, &logTypeAccess, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.LogTypeAccess = logTypeAccess
	}

	role, err := r.client.CreateRole(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Role",
			"Could not create Role, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Created Role", map[string]any{
		"id": role.ID,
	})

	// Update state with response data
	data.ID = types.StringValue(role.ID)
	data.CreatedAt = types.StringValue(role.CreatedAt)
	data.UpdatedAt = types.StringValue(role.UpdatedAt)
	data.Name = types.StringValue(role.Name)

	permList, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Permissions = permList

	if role.LogTypeAccessKind != "" {
		data.LogTypeAccessKind = types.StringValue(role.LogTypeAccessKind)
	}
	if len(role.LogTypeAccess) > 0 {
		logTypeAccessList, diags := types.ListValueFrom(ctx, types.StringType, role.LogTypeAccess)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.LogTypeAccess = logTypeAccessList
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Role",
			fmt.Sprintf("Could not read Role with id %s, unexpected error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Got Role", map[string]any{
		"id": role.ID,
	})

	// Update state with response data
	data.ID = types.StringValue(role.ID)
	data.CreatedAt = types.StringValue(role.CreatedAt)
	data.UpdatedAt = types.StringValue(role.UpdatedAt)
	data.Name = types.StringValue(role.Name)

	permList, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Permissions = permList

	if role.LogTypeAccessKind != "" {
		data.LogTypeAccessKind = types.StringValue(role.LogTypeAccessKind)
	}
	if len(role.LogTypeAccess) > 0 {
		logTypeAccessList, diags := types.ListValueFrom(ctx, types.StringType, role.LogTypeAccess)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.LogTypeAccess = logTypeAccessList
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert permissions list to string slice
	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateRoleInput{
		ID: data.ID.ValueString(),
		RoleModifiableAttributes: client.RoleModifiableAttributes{
			Name:        data.Name.ValueString(),
			Permissions: permissions,
		},
	}

	// Add optional log type access fields if provided
	if !data.LogTypeAccessKind.IsNull() && !data.LogTypeAccessKind.IsUnknown() {
		input.LogTypeAccessKind = data.LogTypeAccessKind.ValueString()
	}
	if !data.LogTypeAccess.IsNull() && !data.LogTypeAccess.IsUnknown() {
		var logTypeAccess []string
		resp.Diagnostics.Append(data.LogTypeAccess.ElementsAs(ctx, &logTypeAccess, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.LogTypeAccess = logTypeAccess
	}

	role, err := r.client.UpdateRole(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Role",
			fmt.Sprintf("Could not update Role with id %s, unexpected error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Updated Role", map[string]any{
		"id": role.ID,
	})

	// Update state with response data
	data.UpdatedAt = types.StringValue(role.UpdatedAt)
	data.Name = types.StringValue(role.Name)

	permList, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Permissions = permList

	if role.LogTypeAccessKind != "" {
		data.LogTypeAccessKind = types.StringValue(role.LogTypeAccessKind)
	}
	if len(role.LogTypeAccess) > 0 {
		logTypeAccessList, diags := types.ListValueFrom(ctx, types.StringType, role.LogTypeAccess)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.LogTypeAccess = logTypeAccessList
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRole(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Role",
			fmt.Sprintf("Could not delete Role with id %s, unexpected error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted Role", map[string]any{
		"id": data.ID.ValueString(),
	})
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
