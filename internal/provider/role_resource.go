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
	"net/http"
	"terraform-provider-panther/internal/client"
	"terraform-provider-panther/internal/provider/resource_role"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const rolePath = "/roles"

var (
	_ resource.Resource                = (*roleResource)(nil)
	_ resource.ResourceWithConfigure   = (*roleResource)(nil)
	_ resource.ResourceWithImportState = (*roleResource)(nil)
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	rest *client.RESTClient
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_role.RoleResourceSchema(ctx)
	resp.Schema.MarkdownDescription = "Represents a Role in Panther"
	applySchemaOverrides(&resp.Schema, []SchemaOverride{
		{Name: "id", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		{Name: "log_type_access_kind", Default: stringdefault.StaticString("")},
	})

	// listdefault for log_type_access — applySchemaOverrides only handles StringAttributes.
	logTypeAccess := resp.Schema.Attributes["log_type_access"].(schema.ListAttribute)
	emptyStringList, _ := types.ListValue(types.StringType, []attr.Value{})
	logTypeAccess.Default = listdefault.StaticValue(emptyStringList)
	resp.Schema.Attributes["log_type_access"] = logTypeAccess
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.RoleInput{
		Name:              data.Name.ValueString(),
		Permissions:       listToStringSlice(ctx, data.Permissions, &resp.Diagnostics),
		LogTypeAccessKind: data.LogTypeAccessKind.ValueString(),
		LogTypeAccess:     listToStringSlice(ctx, data.LogTypeAccess, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := client.RestDo[client.Role](ctx, r.rest, http.MethodPost, rolePath, input)
	if handleCreateError(resp, "Role", err) {
		return
	}

	tflog.Debug(ctx, "Created Role", map[string]any{"id": role.ID})

	data.ID = types.StringValue(role.ID)
	data.CreatedAt = types.StringValue(role.CreatedAt)
	data.UpdatedAt = types.StringValue(role.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := client.RestDo[client.Role](ctx, r.rest, http.MethodGet, rolePath+"/"+data.ID.ValueString(), nil)
	if handleReadError(ctx, resp, "Role", data.ID.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Got Role", map[string]any{"id": role.ID})

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

	data.LogTypeAccessKind = types.StringValue(role.LogTypeAccessKind)

	logTypeAccessList, diags := types.ListValueFrom(ctx, types.StringType, role.LogTypeAccess)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.LogTypeAccess = logTypeAccessList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.RoleInput{
		Name:              data.Name.ValueString(),
		Permissions:       listToStringSlice(ctx, data.Permissions, &resp.Diagnostics),
		LogTypeAccessKind: data.LogTypeAccessKind.ValueString(),
		LogTypeAccess:     listToStringSlice(ctx, data.LogTypeAccess, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := client.RestDo[client.Role](ctx, r.rest, http.MethodPost, rolePath+"/"+data.ID.ValueString(), input)
	if handleUpdateError(ctx, resp, "Role", data.ID.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Updated Role", map[string]any{"id": role.ID})

	data.UpdatedAt = types.StringValue(role.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_role.RoleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, rolePath+"/"+data.ID.ValueString())
	if handleDeleteError(resp, "Role", data.ID.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Deleted Role", map[string]any{"id": data.ID.ValueString()})
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
