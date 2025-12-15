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
	"terraform-provider-panther/internal/provider/resource_simple_rule"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*simpleRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*simpleRuleResource)(nil)
	_ resource.ResourceWithImportState = (*simpleRuleResource)(nil)
)

func NewSimpleRuleResource() resource.Resource {
	return &simpleRuleResource{}
}

type simpleRuleResource struct {
	client client.RestClient
}

func (r *simpleRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_simple_rule"
}

func (r *simpleRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := resource_simple_rule.SimpleRuleResourceSchema(ctx)

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

func (r *simpleRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *simpleRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_simple_rule.SimpleRuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateSimpleRuleInput{
		ID: data.DisplayName.ValueString(),
		SimpleRuleModifiableAttributes: client.SimpleRuleModifiableAttributes{
			DisplayName:        data.DisplayName.ValueString(),
			Detection:          data.Detection.ValueString(),
			Description:        data.Description.ValueString(),
			Severity:           data.Severity.ValueString(),
			Enabled:            data.Enabled.ValueBool(),
			DedupPeriodMinutes: int(data.DedupPeriodMinutes.ValueInt64()),
			Runbook:            data.Runbook.ValueString(),
			Threshold:          int(data.Threshold.ValueInt64()),
			AlertContext:       data.AlertContext.ValueString(),
			AlertTitle:         data.AlertTitle.ValueString(),
			DynamicSeverities:  data.DynamicSeverities.ValueString(),
			GroupBy:            data.GroupBy.ValueString(),
			InlineFilters:      data.InlineFilters.ValueString(),
			PythonBody:         data.PythonBody.ValueString(),
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

	result, err := r.client.CreateSimpleRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create simple_rule, got error: %s", err))
		return
	}

	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Detection = types.StringValue(result.Detection)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(result.DedupPeriodMinutes))
	data.Runbook = types.StringValue(result.Runbook)
	data.Threshold = types.Int64Value(int64(result.Threshold))
	data.AlertContext = types.StringValue(result.AlertContext)
	data.AlertTitle = types.StringValue(result.AlertTitle)
	data.DynamicSeverities = types.StringValue(result.DynamicSeverities)
	data.GroupBy = types.StringValue(result.GroupBy)
	data.InlineFilters = types.StringValue(result.InlineFilters)
	data.PythonBody = types.StringValue(result.PythonBody)
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

	// Set computed fields
	data.CreatedBy = resource_simple_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_simple_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_simple_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})

	tflog.Debug(ctx, "Created SimpleRule", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *simpleRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_simple_rule.SimpleRuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	simpleRuleID := data.Id.ValueString()
	if simpleRuleID == "" {
		simpleRuleID = data.DisplayName.ValueString()
	}
	simpleRule, err := r.client.GetSimpleRule(ctx, simpleRuleID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read simple_rule, got error: %s", err))
		return
	}

	data.Id = types.StringValue(simpleRule.ID)
	data.DisplayName = types.StringValue(simpleRule.DisplayName)
	data.Detection = types.StringValue(simpleRule.Detection)
	data.Description = types.StringValue(simpleRule.Description)
	data.Severity = types.StringValue(simpleRule.Severity)
	data.Enabled = types.BoolValue(simpleRule.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(simpleRule.DedupPeriodMinutes))
	data.Runbook = types.StringValue(simpleRule.Runbook)
	data.Threshold = types.Int64Value(int64(simpleRule.Threshold))
	data.AlertContext = types.StringValue(simpleRule.AlertContext)
	data.AlertTitle = types.StringValue(simpleRule.AlertTitle)
	data.DynamicSeverities = types.StringValue(simpleRule.DynamicSeverities)
	data.GroupBy = types.StringValue(simpleRule.GroupBy)
	data.InlineFilters = types.StringValue(simpleRule.InlineFilters)
	data.PythonBody = types.StringValue(simpleRule.PythonBody)
	data.CreatedAt = types.StringValue(simpleRule.CreatedAt)
	data.LastModified = types.StringValue(simpleRule.UpdatedAt)

	// Convert log types back to list
	if len(simpleRule.LogTypes) > 0 {
		elements := make([]types.String, len(simpleRule.LogTypes))
		for i, logType := range simpleRule.LogTypes {
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

	// Handle tags with order preservation
	if len(simpleRule.Tags) > 0 {
		currentTags := make([]string, 0)
		if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
			for _, elem := range data.Tags.Elements() {
				if strVal, ok := elem.(types.String); ok {
					currentTags = append(currentTags, strVal.ValueString())
				}
			}
		}

		tagsChanged := len(currentTags) != len(simpleRule.Tags)
		if !tagsChanged {
			apiTagsMap := make(map[string]bool)
			for _, tag := range simpleRule.Tags {
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
			elements := make([]types.String, len(simpleRule.Tags))
			for i, tag := range simpleRule.Tags {
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

	// Set computed fields
	data.CreatedBy = resource_simple_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_simple_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_simple_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *simpleRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_simple_rule.SimpleRuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateSimpleRuleInput{
		ID: data.Id.ValueString(),
		SimpleRuleModifiableAttributes: client.SimpleRuleModifiableAttributes{
			DisplayName:        data.DisplayName.ValueString(),
			Detection:          data.Detection.ValueString(),
			Description:        data.Description.ValueString(),
			Severity:           data.Severity.ValueString(),
			Enabled:            data.Enabled.ValueBool(),
			DedupPeriodMinutes: int(data.DedupPeriodMinutes.ValueInt64()),
			Runbook:            data.Runbook.ValueString(),
			Threshold:          int(data.Threshold.ValueInt64()),
			AlertContext:       data.AlertContext.ValueString(),
			AlertTitle:         data.AlertTitle.ValueString(),
			DynamicSeverities:  data.DynamicSeverities.ValueString(),
			GroupBy:            data.GroupBy.ValueString(),
			InlineFilters:      data.InlineFilters.ValueString(),
			PythonBody:         data.PythonBody.ValueString(),
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

	result, err := r.client.UpdateSimpleRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update simple_rule, got error: %s", err))
		return
	}

	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Detection = types.StringValue(result.Detection)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(result.DedupPeriodMinutes))
	data.Runbook = types.StringValue(result.Runbook)
	data.Threshold = types.Int64Value(int64(result.Threshold))
	data.AlertContext = types.StringValue(result.AlertContext)
	data.AlertTitle = types.StringValue(result.AlertTitle)
	data.DynamicSeverities = types.StringValue(result.DynamicSeverities)
	data.GroupBy = types.StringValue(result.GroupBy)
	data.InlineFilters = types.StringValue(result.InlineFilters)
	data.PythonBody = types.StringValue(result.PythonBody)
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

	// Set computed fields
	data.CreatedBy = resource_simple_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_simple_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_simple_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})

	tflog.Debug(ctx, "Updated SimpleRule", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *simpleRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_simple_rule.SimpleRuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSimpleRule(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete simple_rule, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleted SimpleRule", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *simpleRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
