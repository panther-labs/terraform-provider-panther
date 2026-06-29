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
	"terraform-provider-panther/internal/provider/resource_policy"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const policyPath = "/policies"

var (
	_ resource.Resource                = (*policyResource)(nil)
	_ resource.ResourceWithConfigure   = (*policyResource)(nil)
	_ resource.ResourceWithImportState = (*policyResource)(nil)
)

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

type policyResource struct {
	rest *client.RESTClient
}

func (r *policyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := resource_policy.PolicyResourceSchema(ctx)

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

func (r *policyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_policy.PolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.PolicyInput{
		ID:          data.DisplayName.ValueString(),
		DisplayName: data.DisplayName.ValueString(),
		Body:        data.Body.ValueString(),
		Description: data.Description.ValueString(),
		Severity:    data.Severity.ValueString(),
		Enabled:     data.Enabled.ValueBool(),
	}

	if !data.ResourceTypes.IsNull() && !data.ResourceTypes.IsUnknown() {
		resourceTypes := make([]string, 0, len(data.ResourceTypes.Elements()))
		for _, elem := range data.ResourceTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				resourceTypes = append(resourceTypes, strVal.ValueString())
			}
		}
		input.ResourceTypes = resourceTypes
	}

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make([]string, 0, len(data.Tags.Elements()))
		for _, elem := range data.Tags.Elements() {
			if strVal, ok := elem.(types.String); ok {
				tags = append(tags, strVal.ValueString())
			}
		}
		input.Tags = tags
	}

	result, err := client.RestDo[client.Policy](ctx, r.rest, http.MethodPost, policyPath, input)
	if handleCreateError(resp, "Policy", err) {
		return
	}

	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.LastModified)

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

	policyID := data.Id.ValueString()
	if policyID == "" {
		policyID = data.DisplayName.ValueString()
	}

	result, err := client.RestDo[client.Policy](ctx, r.rest, http.MethodGet, policyPath+"/"+policyID, nil)
	if handleReadError(ctx, resp, "Policy", policyID, err) {
		return
	}

	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.LastModified)

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

	// Preserve tag order: only update tags if content changed.
	if len(result.Tags) > 0 {
		currentTags := make([]string, 0)
		if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
			for _, elem := range data.Tags.Elements() {
				if strVal, ok := elem.(types.String); ok {
					currentTags = append(currentTags, strVal.ValueString())
				}
			}
		}

		tagsChanged := len(currentTags) != len(result.Tags)
		if !tagsChanged {
			apiTagsMap := make(map[string]bool)
			for _, tag := range result.Tags {
				apiTagsMap[tag] = true
			}
			for _, tag := range currentTags {
				if !apiTagsMap[tag] {
					tagsChanged = true
					break
				}
			}
		}

		if tagsChanged {
			elements := make([]types.String, len(result.Tags))
			for i, tag := range result.Tags {
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
		data.Tags = types.ListNull(types.StringType)
	}

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

	input := client.PolicyInput{
		ID:          data.Id.ValueString(),
		DisplayName: data.DisplayName.ValueString(),
		Body:        data.Body.ValueString(),
		Description: data.Description.ValueString(),
		Severity:    data.Severity.ValueString(),
		Enabled:     data.Enabled.ValueBool(),
	}

	if !data.ResourceTypes.IsNull() && !data.ResourceTypes.IsUnknown() {
		resourceTypes := make([]string, 0, len(data.ResourceTypes.Elements()))
		for _, elem := range data.ResourceTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				resourceTypes = append(resourceTypes, strVal.ValueString())
			}
		}
		input.ResourceTypes = resourceTypes
	}

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make([]string, 0, len(data.Tags.Elements()))
		for _, elem := range data.Tags.Elements() {
			if strVal, ok := elem.(types.String); ok {
				tags = append(tags, strVal.ValueString())
			}
		}
		input.Tags = tags
	}

	result, err := client.RestDo[client.Policy](ctx, r.rest, http.MethodPut, policyPath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(ctx, resp, "Policy", data.Id.ValueString(), err) {
		return
	}

	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.LastModified)

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

	err := client.RestDelete(ctx, r.rest, policyPath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "Policy", data.Id.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Deleted Policy", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
