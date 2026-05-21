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
	"terraform-provider-panther/internal/provider/resource_simple_rule"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const simpleRulePath = "/simple-rules"

var (
	_ resource.Resource                = (*simpleRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*simpleRuleResource)(nil)
	_ resource.ResourceWithImportState = (*simpleRuleResource)(nil)
)

func NewSimpleRuleResource() resource.Resource {
	return &simpleRuleResource{}
}

type simpleRuleResource struct {
	rest *client.RESTClient
}

func (r *simpleRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
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

func (r *simpleRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *simpleRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_simple_rule.SimpleRuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.SimpleRuleInput{
		ID:                 data.DisplayName.ValueString(),
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
	}

	if !data.LogTypes.IsNull() && !data.LogTypes.IsUnknown() {
		logTypes := make([]string, 0, len(data.LogTypes.Elements()))
		for _, elem := range data.LogTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				logTypes = append(logTypes, strVal.ValueString())
			}
		}
		input.LogTypes = logTypes
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

	result, err := client.RestDo[client.SimpleRule](ctx, r.rest, http.MethodPost, simpleRulePath, input)
	if handleCreateError(resp, "SimpleRule", err) {
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
	data.LastModified = types.StringValue(result.LastModified)

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

	result, err := client.RestDo[client.SimpleRule](ctx, r.rest, http.MethodGet, simpleRulePath+"/"+simpleRuleID, nil)
	if handleReadError(ctx, resp, "SimpleRule", simpleRuleID, err) {
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
	data.LastModified = types.StringValue(result.LastModified)

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

	input := client.SimpleRuleInput{
		ID:                 data.Id.ValueString(),
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
	}

	if !data.LogTypes.IsNull() && !data.LogTypes.IsUnknown() {
		logTypes := make([]string, 0, len(data.LogTypes.Elements()))
		for _, elem := range data.LogTypes.Elements() {
			if strVal, ok := elem.(types.String); ok {
				logTypes = append(logTypes, strVal.ValueString())
			}
		}
		input.LogTypes = logTypes
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

	result, err := client.RestDo[client.SimpleRule](ctx, r.rest, http.MethodPut, simpleRulePath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(ctx, resp, "SimpleRule", data.Id.ValueString(), err) {
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
	data.LastModified = types.StringValue(result.LastModified)

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

	err := client.RestDelete(ctx, r.rest, simpleRulePath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "SimpleRule", data.Id.ValueString(), err) {
		return
	}

	tflog.Debug(ctx, "Deleted SimpleRule", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *simpleRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
