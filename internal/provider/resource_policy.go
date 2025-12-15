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
	"terraform-provider-panther/internal/provider/resource_policy"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*policyResource)(nil)
	_ resource.ResourceWithConfigure   = (*policyResource)(nil)
	_ resource.ResourceWithImportState = (*policyResource)(nil)
)

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

type policyResource struct {
	client client.RestClient
}

func (r *policyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Use the generated schema
	generatedSchema := resource_policy.PolicyResourceSchema(ctx)

	// Add the ID field with UseStateForUnknown as required by limitations
	if generatedSchema.Attributes == nil {
		generatedSchema.Attributes = make(map[string]schema.Attribute)
	}

	generatedSchema.Attributes["id"] = schema.StringAttribute{
		Computed: true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	resp.Schema = generatedSchema
}

func (r *policyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*panther.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *panther.APIClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = apiClient.RestClient
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_policy.PolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert from generated model to our client types
	input := client.CreatePolicyInput{
		ID: data.DisplayName.ValueString(), // Using display_name as ID
		PolicyModifiableAttributes: client.PolicyModifiableAttributes{
			DisplayName:   data.DisplayName.ValueString(),
			Body:          data.Body.ValueString(),
			Description:   data.Description.ValueString(),
			Severity:      data.Severity.ValueString(),
			Enabled:       data.Enabled.ValueBool(),
		},
	}

	// Convert resource types
	if !data.ResourceTypes.IsNull() && !data.ResourceTypes.IsUnknown() {
		resourceTypes := make([]string, 0, len(data.ResourceTypes.Elements()))
		for _, elem := range data.ResourceTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				resourceTypes = append(resourceTypes, strVal.ValueString())
			}
		}
		input.ResourceTypes = resourceTypes
	}

	// Convert tags
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make([]string, 0, len(data.Tags.Elements()))
		for _, elem := range data.Tags.Elements() {
			if strVal, ok := elem.(types.String); ok {
				tags = append(tags, strVal.ValueString())
			}
		}
		input.Tags = tags
	}

	result, err := r.client.CreatePolicy(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy, got error: %s", err))
		return
	}

	// Update the model with the result
	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.UpdatedAt)

	// Convert resource types back to list
	if len(result.ResourceTypes) > 0 {
		elements := make([]types.String, len(result.ResourceTypes))
		for i, resourceType := range result.ResourceTypes {
			elements[i] = types.StringValue(resourceType)
		}
		resourceTypesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceTypes = resourceTypesList
	} else {
		data.ResourceTypes = types.ListNull(types.StringType)
	}

	// Set other computed fields to null/empty for now
	data.CreatedBy = resource_policy.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.Tests = types.ListNull(resource_policy.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_policy.TestsValue{}.AttributeTypes(ctx),
		},
	})
	data.Suppressions = types.ListNull(types.StringType)

	tflog.Debug(ctx, "Created Policy", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_policy.PolicyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use ID if available, otherwise fall back to DisplayName for backward compatibility
	policyID := data.Id.ValueString()
	if policyID == "" {
		policyID = data.DisplayName.ValueString()
	}
	policy, err := r.client.GetPolicy(ctx, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	// Update model with API response
	data.Id = types.StringValue(policy.ID)
	data.DisplayName = types.StringValue(policy.DisplayName)
	data.Body = types.StringValue(policy.Body)
	data.Description = types.StringValue(policy.Description)
	data.Severity = types.StringValue(policy.Severity)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.CreatedAt = types.StringValue(policy.CreatedAt)
	data.LastModified = types.StringValue(policy.UpdatedAt)

	// Convert resource types back to list
	if len(policy.ResourceTypes) > 0 {
		elements := make([]types.String, len(policy.ResourceTypes))
		for i, resourceType := range policy.ResourceTypes {
			elements[i] = types.StringValue(resourceType)
		}
		resourceTypesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceTypes = resourceTypesList
	} else {
		data.ResourceTypes = types.ListNull(types.StringType)
	}

	// Only update tags if they have actually changed (content-wise, ignoring order)
	if len(policy.Tags) > 0 {
		// Get current tags from state
		currentTags := make([]string, 0)
		if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
			for _, elem := range data.Tags.Elements() {
				if strVal, ok := elem.(types.String); ok {
					currentTags = append(currentTags, strVal.ValueString())
				}
			}
		}

		// Check if tag content has changed (ignore order)
		tagsChanged := len(currentTags) != len(policy.Tags)
		if !tagsChanged {
			apiTagsMap := make(map[string]bool)
			for _, tag := range policy.Tags {
				apiTagsMap[tag] = true
			}
			for _, tag := range currentTags {
				if !apiTagsMap[tag] {
					tagsChanged = true
					break
				}
			}
		}

		// Only update tags if content changed, preserving existing order
		if tagsChanged {
			elements := make([]types.String, len(policy.Tags))
			for i, tag := range policy.Tags {
				elements[i] = types.StringValue(tag)
			}
			tagsList, diags := types.ListValueFrom(ctx, types.StringType, elements)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			data.Tags = tagsList
		}
	} else if !data.Tags.IsNull() {
		// API returned no tags but state has tags - clear them
		data.Tags = types.ListNull(types.StringType)
	}

	// Set other computed fields to null/empty for now
	data.CreatedBy = resource_policy.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.Tests = types.ListNull(resource_policy.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_policy.TestsValue{}.AttributeTypes(ctx),
		},
	})
	data.Suppressions = types.ListNull(types.StringType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_policy.PolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdatePolicyInput{
		ID: data.Id.ValueString(),
		PolicyModifiableAttributes: client.PolicyModifiableAttributes{
			DisplayName:   data.DisplayName.ValueString(),
			Body:          data.Body.ValueString(),
			Description:   data.Description.ValueString(),
			Severity:      data.Severity.ValueString(),
			Enabled:       data.Enabled.ValueBool(),
		},
	}

	// Convert resource types
	if !data.ResourceTypes.IsNull() && !data.ResourceTypes.IsUnknown() {
		resourceTypes := make([]string, 0, len(data.ResourceTypes.Elements()))
		for _, elem := range data.ResourceTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				resourceTypes = append(resourceTypes, strVal.ValueString())
			}
		}
		input.ResourceTypes = resourceTypes
	}

	// Convert tags
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make([]string, 0, len(data.Tags.Elements()))
		for _, elem := range data.Tags.Elements() {
			if strVal, ok := elem.(types.String); ok {
				tags = append(tags, strVal.ValueString())
			}
		}
		input.Tags = tags
	}

	result, err := r.client.UpdatePolicy(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update policy, got error: %s", err))
		return
	}

	// Update the model with the result
	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.UpdatedAt)

	// Convert resource types back to list
	if len(result.ResourceTypes) > 0 {
		elements := make([]types.String, len(result.ResourceTypes))
		for i, resourceType := range result.ResourceTypes {
			elements[i] = types.StringValue(resourceType)
		}
		resourceTypesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceTypes = resourceTypesList
	} else {
		data.ResourceTypes = types.ListNull(types.StringType)
	}

	// Set other computed fields to null/empty for now
	data.CreatedBy = resource_policy.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.Tests = types.ListNull(resource_policy.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_policy.TestsValue{}.AttributeTypes(ctx),
		},
	})
	data.Suppressions = types.ListNull(types.StringType)

	tflog.Debug(ctx, "Updated Policy", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_policy.PolicyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePolicy(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleted Policy", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
