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
	"terraform-provider-panther/internal/provider/resource_user"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*userResource)(nil)
	_ resource.ResourceWithConfigure   = (*userResource)(nil)
	_ resource.ResourceWithImportState = (*userResource)(nil)
)

func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResource struct {
	client client.RestClient
}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_user.UserResourceSchema(ctx)
	// Add UseStateForUnknown plan modifier to the id attribute
	idAttr := resp.Schema.Attributes["id"].(schema.StringAttribute)
	idAttr.PlanModifiers = append(idAttr.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	resp.Schema.Attributes["id"] = idAttr
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert role reference to API format
	roleRef := client.UserRoleRef{}
	if !data.Role.ID.IsNull() && !data.Role.ID.IsUnknown() {
		roleRef.ID = data.Role.ID.ValueString()
	}
	if !data.Role.Name.IsNull() && !data.Role.Name.IsUnknown() {
		roleRef.Name = data.Role.Name.ValueString()
	}

	input := client.CreateUserInput{
		UserModifiableAttributes: client.UserModifiableAttributes{
			Email:      data.Email.ValueString(),
			GivenName:  data.GivenName.ValueString(),
			FamilyName: data.FamilyName.ValueString(),
			Role:       roleRef,
		},
	}

	user, err := r.client.CreateUser(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating User",
			"Could not create User, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Created User", map[string]any{
		"id": user.ID,
	})

	// Update state with response data
	data.ID = types.StringValue(user.ID)
	data.CreatedAt = types.StringValue(user.CreatedAt)
	data.Enabled = types.BoolValue(user.Enabled)
	data.Status = types.StringValue(user.Status)
	if user.LastLoggedInAt != "" {
		data.LastLoggedInAt = types.StringValue(user.LastLoggedInAt)
	}

	// Update role from response
	data.Role = resource_user.RoleValue{
		ID:   types.StringValue(user.Role.ID),
		Name: types.StringValue(user.Role.Name),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading User",
			fmt.Sprintf("Could not read User with id %s, unexpected error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Got User", map[string]any{
		"id": user.ID,
	})

	// Update state with response data
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

	data.Role = resource_user.RoleValue{
		ID:   types.StringValue(user.Role.ID),
		Name: types.StringValue(user.Role.Name),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert role reference to API format
	roleRef := client.UserRoleRef{}
	if !data.Role.ID.IsNull() && !data.Role.ID.IsUnknown() {
		roleRef.ID = data.Role.ID.ValueString()
	}
	if !data.Role.Name.IsNull() && !data.Role.Name.IsUnknown() {
		roleRef.Name = data.Role.Name.ValueString()
	}

	input := client.UpdateUserInput{
		ID: data.ID.ValueString(),
		UserModifiableAttributes: client.UserModifiableAttributes{
			Email:      data.Email.ValueString(),
			GivenName:  data.GivenName.ValueString(),
			FamilyName: data.FamilyName.ValueString(),
			Role:       roleRef,
		},
	}

	user, err := r.client.UpdateUser(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating User",
			fmt.Sprintf("Could not update User with id %s, unexpected error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Updated User", map[string]any{
		"id": user.ID,
	})

	// Update state with response data
	data.Email = types.StringValue(user.Email)
	data.GivenName = types.StringValue(user.GivenName)
	data.FamilyName = types.StringValue(user.FamilyName)
	data.Enabled = types.BoolValue(user.Enabled)
	data.Status = types.StringValue(user.Status)
	if user.LastLoggedInAt != "" {
		data.LastLoggedInAt = types.StringValue(user.LastLoggedInAt)
	}

	data.Role = resource_user.RoleValue{
		ID:   types.StringValue(user.Role.ID),
		Name: types.StringValue(user.Role.Name),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_user.UserModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUser(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting User",
			fmt.Sprintf("Could not delete User with id %s, unexpected error: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted User", map[string]any{
		"id": data.ID.ValueString(),
	})
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
