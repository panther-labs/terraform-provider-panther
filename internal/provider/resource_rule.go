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
	"terraform-provider-panther/internal/provider/resource_rule"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*ruleResource)(nil)
	_ resource.ResourceWithConfigure   = (*ruleResource)(nil)
	_ resource.ResourceWithImportState = (*ruleResource)(nil)
)

func NewRuleResource() resource.Resource {
	return &ruleResource{}
}

type ruleResource struct {
	client client.RestClient
}

func (r *ruleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

func (r *ruleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Use the generated schema
	generatedSchema := resource_rule.RuleResourceSchema(ctx)
	
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

func (r *ruleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_rule.RuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert from generated model to our client types
	input := client.CreateRuleInput{
		ID: data.DisplayName.ValueString(), // Using display_name as ID
		RuleModifiableAttributes: client.RuleModifiableAttributes{
			DisplayName:        data.DisplayName.ValueString(),
			Body:               data.Body.ValueString(),
			Description:        data.Description.ValueString(),
			Severity:           data.Severity.ValueString(),
			Enabled:            data.Enabled.ValueBool(),
			DedupPeriodMinutes: int(data.DedupPeriodMinutes.ValueInt64()),
			Runbook:            data.Runbook.ValueString(),
		},
	}

	// Convert log types
	if !data.LogTypes.IsNull() && !data.LogTypes.IsUnknown() {
		logTypes := make([]string, 0, len(data.LogTypes.Elements()))
		for _, elem := range data.LogTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				logTypes = append(logTypes, strVal.ValueString())
			}
		}
		input.LogTypes = logTypes
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

	result, err := r.client.CreateRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create rule, got error: %s", err))
		return
	}

	// Update the model with the result - populate all fields including computed ones
	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(result.DedupPeriodMinutes))
	data.Runbook = types.StringValue(result.Runbook)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.UpdatedAt)
	
	// Convert log types back to list
	if len(result.LogTypes) > 0 {
		elements := make([]types.String, len(result.LogTypes))
		for i, logType := range result.LogTypes {
			elements[i] = types.StringValue(logType)
		}
		logTypesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.LogTypes = logTypesList
	} else {
		data.LogTypes = types.ListNull(types.StringType)
	}
	
	// Preserve original tag order from the plan (Terraform expects consistent ordering)
	// The data.Tags already has the correct values from the plan, so we don't need to overwrite it
	
	// Set other computed fields to null/empty for now since they're not returned by the API
	data.CreatedBy = resource_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.InlineFilters = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})
	data.Threshold = types.Int64Value(1) // Use default value

	tflog.Debug(ctx, "Created Rule", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule.RuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use ID if available, otherwise fall back to DisplayName for backward compatibility
	ruleID := data.Id.ValueString()
	if ruleID == "" {
		ruleID = data.DisplayName.ValueString()
	}
	rule, err := r.client.GetRule(ctx, ruleID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read rule, got error: %s", err))
		return
	}

	// Update model with API response - populate all fields including computed ones
	data.Id = types.StringValue(rule.ID)
	data.DisplayName = types.StringValue(rule.DisplayName)
	data.Body = types.StringValue(rule.Body)
	data.Description = types.StringValue(rule.Description)
	data.Severity = types.StringValue(rule.Severity)
	data.Enabled = types.BoolValue(rule.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(rule.DedupPeriodMinutes))
	data.Runbook = types.StringValue(rule.Runbook)
	data.CreatedAt = types.StringValue(rule.CreatedAt)
	data.LastModified = types.StringValue(rule.UpdatedAt)
	
	// Convert log types back to list
	if len(rule.LogTypes) > 0 {
		elements := make([]types.String, len(rule.LogTypes))
		for i, logType := range rule.LogTypes {
			elements[i] = types.StringValue(logType)
		}
		logTypesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.LogTypes = logTypesList
	} else {
		data.LogTypes = types.ListNull(types.StringType)
	}
	
	// Only update tags if they have actually changed (content-wise, ignoring order)
	// to preserve the order from the original plan/state
	if len(rule.Tags) > 0 {
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
		tagsChanged := len(currentTags) != len(rule.Tags)
		if !tagsChanged {
			apiTagsMap := make(map[string]bool)
			for _, tag := range rule.Tags {
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
			elements := make([]types.String, len(rule.Tags))
			for i, tag := range rule.Tags {
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
	
	// Set other computed fields to null/empty for now since they're not returned by the API
	data.CreatedBy = resource_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.InlineFilters = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})
	data.Threshold = types.Int64Value(1) // Use default value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_rule.RuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateRuleInput{
		ID: data.Id.ValueString(),
		RuleModifiableAttributes: client.RuleModifiableAttributes{
			DisplayName:        data.DisplayName.ValueString(),
			Body:               data.Body.ValueString(),
			Description:        data.Description.ValueString(),
			Severity:           data.Severity.ValueString(),
			Enabled:            data.Enabled.ValueBool(),
			DedupPeriodMinutes: int(data.DedupPeriodMinutes.ValueInt64()),
			Runbook:            data.Runbook.ValueString(),
		},
	}

	// Convert log types
	if !data.LogTypes.IsNull() && !data.LogTypes.IsUnknown() {
		logTypes := make([]string, 0, len(data.LogTypes.Elements()))
		for _, elem := range data.LogTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				logTypes = append(logTypes, strVal.ValueString())
			}
		}
		input.LogTypes = logTypes
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

	result, err := r.client.UpdateRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update rule, got error: %s", err))
		return
	}

	// Update the model with the result - populate all fields including computed ones
	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(result.DedupPeriodMinutes))
	data.Runbook = types.StringValue(result.Runbook)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.UpdatedAt)
	
	// Convert log types back to list
	if len(result.LogTypes) > 0 {
		elements := make([]types.String, len(result.LogTypes))
		for i, logType := range result.LogTypes {
			elements[i] = types.StringValue(logType)
		}
		logTypesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.LogTypes = logTypesList
	} else {
		data.LogTypes = types.ListNull(types.StringType)
	}
	
	// Preserve original tag order from the plan (Terraform expects consistent ordering)
	// The data.Tags already has the correct values from the plan, so we don't need to overwrite it
	
	// Set other computed fields to null/empty for now since they're not returned by the API
	data.CreatedBy = resource_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.InlineFilters = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})
	data.Threshold = types.Int64Value(1) // Use default value

	tflog.Debug(ctx, "Updated Rule", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_rule.RuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRule(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete rule, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleted Rule", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *ruleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}