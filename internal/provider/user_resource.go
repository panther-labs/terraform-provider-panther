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
	"terraform-provider-panther/internal/provider/resource_user"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const userPath = "/users"

var (
	_ resource.Resource                = (*userResource)(nil)
	_ resource.ResourceWithConfigure   = (*userResource)(nil)
	_ resource.ResourceWithImportState = (*userResource)(nil)
)

func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResource struct {
	rest *client.RESTClient
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_user.UserResourceSchema(ctx)
	resp.Schema.MarkdownDescription = "Represents a User in Panther"
	applySchemaOverrides(&resp.Schema, []SchemaOverride{
		{Name: "id", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
	})
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleRef := client.UserRoleRef{}
	if !data.Role.ID.IsNull() && !data.Role.ID.IsUnknown() && data.Role.ID.ValueString() != "" {
		roleRef.ID = data.Role.ID.ValueString()
	}
	if !data.Role.Name.IsNull() && !data.Role.Name.IsUnknown() && data.Role.Name.ValueString() != "" {
		roleRef.Name = data.Role.Name.ValueString()
	}

	input := client.UserInput{
		Email:      data.Email.ValueString(),
		GivenName:  data.GivenName.ValueString(),
		FamilyName: data.FamilyName.ValueString(),
		Role:       roleRef,
	}

	user, err := client.RestDo[client.User](ctx, r.rest, http.MethodPost, userPath, input)
	if handleCreateError(resp, "User", err) {
		return
	}

	tflog.Debug(ctx, "Created User", map[string]any{"id": user.ID})

	data.ID = types.StringValue(user.ID)
	data.CreatedAt = types.StringValue(user.CreatedAt)
	data.Enabled = types.BoolValue(user.Enabled)
	data.Status = types.StringValue(user.Status)
	if user.LastLoggedInAt != "" {
		data.LastLoggedInAt = types.StringValue(user.LastLoggedInAt)
	}

	// Only overwrite role fields if the API returned a non-empty value.
	if user.Role.ID != "" {
		data.Role.ID = types.StringValue(user.Role.ID)
	}
	if user.Role.Name != "" {
		data.Role.Name = types.StringValue(user.Role.Name)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := client.RestDo[client.User](ctx, r.rest, http.MethodGet, userPath+"/"+data.ID.ValueString(), nil)
	if handleReadError(ctx, resp, "User", data.ID.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Got User", map[string]any{"id": user.ID})

	data.ID = types.StringValue(user.ID)
	data.Email = types.StringValue(user.Email)
	data.GivenName = types.StringValue(user.GivenName)
	data.FamilyName = types.StringValue(user.FamilyName)
	data.CreatedAt = types.StringValue(user.CreatedAt)
	data.Enabled = types.BoolValue(user.Enabled)
	data.Status = types.StringValue(user.Status)
	if user.LastLoggedInAt != "" {
		data.LastLoggedInAt = types.StringValue(user.LastLoggedInAt)
	}

	// Only overwrite role fields if the API returned a non-empty value.
	if user.Role.ID != "" {
		data.Role.ID = types.StringValue(user.Role.ID)
	}
	if user.Role.Name != "" {
		data.Role.Name = types.StringValue(user.Role.Name)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleRef := client.UserRoleRef{}
	if !data.Role.ID.IsNull() && !data.Role.ID.IsUnknown() && data.Role.ID.ValueString() != "" {
		roleRef.ID = data.Role.ID.ValueString()
	}
	if !data.Role.Name.IsNull() && !data.Role.Name.IsUnknown() && data.Role.Name.ValueString() != "" {
		roleRef.Name = data.Role.Name.ValueString()
	}

	input := client.UserInput{
		Email:      data.Email.ValueString(),
		GivenName:  data.GivenName.ValueString(),
		FamilyName: data.FamilyName.ValueString(),
		Role:       roleRef,
	}

	user, err := client.RestDo[client.User](ctx, r.rest, http.MethodPost, userPath+"/"+data.ID.ValueString(), input)
	if handleUpdateError(ctx, resp, "User", data.ID.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Updated User", map[string]any{"id": user.ID})

	// Only overwrite role fields if the API returned a non-empty value.
	if user.Role.ID != "" {
		data.Role.ID = types.StringValue(user.Role.ID)
	}
	if user.Role.Name != "" {
		data.Role.Name = types.StringValue(user.Role.Name)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, userPath+"/"+data.ID.ValueString())
	if handleDeleteError(resp, "User", data.ID.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Deleted User", map[string]any{"id": data.ID.ValueString()})
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
