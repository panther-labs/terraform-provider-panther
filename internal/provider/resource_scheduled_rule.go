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
	"terraform-provider-panther/internal/provider/resource_scheduled_rule"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*scheduledRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*scheduledRuleResource)(nil)
	_ resource.ResourceWithImportState = (*scheduledRuleResource)(nil)
)

func NewScheduledRuleResource() resource.Resource {
	return &scheduledRuleResource{}
}

type scheduledRuleResource struct {
	client client.RestClient
}

func (r *scheduledRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_rule"
}

func (r *scheduledRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := resource_scheduled_rule.ScheduledRuleResourceSchema(ctx)

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

func (r *scheduledRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *scheduledRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_scheduled_rule.ScheduledRuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateScheduledRuleInput{
		ID: data.DisplayName.ValueString(),
		ScheduledRuleModifiableAttributes: client.ScheduledRuleModifiableAttributes{
			DisplayName:        data.DisplayName.ValueString(),
			Body:               data.Body.ValueString(),
			Description:        data.Description.ValueString(),
			Severity:           data.Severity.ValueString(),
			Enabled:            data.Enabled.ValueBool(),
			DedupPeriodMinutes: int(data.DedupPeriodMinutes.ValueInt64()),
			Runbook:            data.Runbook.ValueString(),
			Threshold:          int(data.Threshold.ValueInt64()),
		},
	}

	// Convert scheduled queries
	if !data.ScheduledQueries.IsNull() && !data.ScheduledQueries.IsUnknown() {
		scheduledQueries := make([]string, 0, len(data.ScheduledQueries.Elements()))
		for _, elem := range data.ScheduledQueries.Elements() {
			if strVal, ok := elem.(types.String); ok {
				scheduledQueries = append(scheduledQueries, strVal.ValueString())
			}
		}
		input.ScheduledQueries = scheduledQueries
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

	result, err := r.client.CreateScheduledRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create scheduled_rule, got error: %s", err))
		return
	}

	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(result.DedupPeriodMinutes))
	data.Runbook = types.StringValue(result.Runbook)
	data.Threshold = types.Int64Value(int64(result.Threshold))
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.UpdatedAt)

	// Convert scheduled queries back to list
	if len(result.ScheduledQueries) > 0 {
		elements := make([]types.String, len(result.ScheduledQueries))
		for i, query := range result.ScheduledQueries {
			elements[i] = types.StringValue(query)
		}
		queriesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ScheduledQueries = queriesList
	} else {
		data.ScheduledQueries = types.ListNull(types.StringType)
	}

	// Set computed fields
	data.CreatedBy = resource_scheduled_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_scheduled_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_scheduled_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})

	tflog.Debug(ctx, "Created ScheduledRule", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *scheduledRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_scheduled_rule.ScheduledRuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scheduledRuleID := data.Id.ValueString()
	if scheduledRuleID == "" {
		scheduledRuleID = data.DisplayName.ValueString()
	}
	scheduledRule, err := r.client.GetScheduledRule(ctx, scheduledRuleID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled_rule, got error: %s", err))
		return
	}

	data.Id = types.StringValue(scheduledRule.ID)
	data.DisplayName = types.StringValue(scheduledRule.DisplayName)
	data.Body = types.StringValue(scheduledRule.Body)
	data.Description = types.StringValue(scheduledRule.Description)
	data.Severity = types.StringValue(scheduledRule.Severity)
	data.Enabled = types.BoolValue(scheduledRule.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(scheduledRule.DedupPeriodMinutes))
	data.Runbook = types.StringValue(scheduledRule.Runbook)
	data.Threshold = types.Int64Value(int64(scheduledRule.Threshold))
	data.CreatedAt = types.StringValue(scheduledRule.CreatedAt)
	data.LastModified = types.StringValue(scheduledRule.UpdatedAt)

	// Convert scheduled queries back to list
	if len(scheduledRule.ScheduledQueries) > 0 {
		elements := make([]types.String, len(scheduledRule.ScheduledQueries))
		for i, query := range scheduledRule.ScheduledQueries {
			elements[i] = types.StringValue(query)
		}
		queriesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ScheduledQueries = queriesList
	} else {
		data.ScheduledQueries = types.ListNull(types.StringType)
	}

	// Handle tags with order preservation
	if len(scheduledRule.Tags) > 0 {
		currentTags := make([]string, 0)
		if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
			for _, elem := range data.Tags.Elements() {
				if strVal, ok := elem.(types.String); ok {
					currentTags = append(currentTags, strVal.ValueString())
				}
			}
		}

		tagsChanged := len(currentTags) != len(scheduledRule.Tags)
		if !tagsChanged {
			apiTagsMap := make(map[string]bool)
			for _, tag := range scheduledRule.Tags {
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
			elements := make([]types.String, len(scheduledRule.Tags))
			for i, tag := range scheduledRule.Tags {
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
	data.CreatedBy = resource_scheduled_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_scheduled_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_scheduled_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *scheduledRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_scheduled_rule.ScheduledRuleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateScheduledRuleInput{
		ID: data.Id.ValueString(),
		ScheduledRuleModifiableAttributes: client.ScheduledRuleModifiableAttributes{
			DisplayName:        data.DisplayName.ValueString(),
			Body:               data.Body.ValueString(),
			Description:        data.Description.ValueString(),
			Severity:           data.Severity.ValueString(),
			Enabled:            data.Enabled.ValueBool(),
			DedupPeriodMinutes: int(data.DedupPeriodMinutes.ValueInt64()),
			Runbook:            data.Runbook.ValueString(),
			Threshold:          int(data.Threshold.ValueInt64()),
		},
	}

	// Convert scheduled queries
	if !data.ScheduledQueries.IsNull() && !data.ScheduledQueries.IsUnknown() {
		scheduledQueries := make([]string, 0, len(data.ScheduledQueries.Elements()))
		for _, elem := range data.ScheduledQueries.Elements() {
			if strVal, ok := elem.(types.String); ok {
				scheduledQueries = append(scheduledQueries, strVal.ValueString())
			}
		}
		input.ScheduledQueries = scheduledQueries
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

	result, err := r.client.UpdateScheduledRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update scheduled_rule, got error: %s", err))
		return
	}

	data.Id = types.StringValue(result.ID)
	data.DisplayName = types.StringValue(result.DisplayName)
	data.Body = types.StringValue(result.Body)
	data.Description = types.StringValue(result.Description)
	data.Severity = types.StringValue(result.Severity)
	data.Enabled = types.BoolValue(result.Enabled)
	data.DedupPeriodMinutes = types.Int64Value(int64(result.DedupPeriodMinutes))
	data.Runbook = types.StringValue(result.Runbook)
	data.Threshold = types.Int64Value(int64(result.Threshold))
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.LastModified = types.StringValue(result.UpdatedAt)

	// Convert scheduled queries back to list
	if len(result.ScheduledQueries) > 0 {
		elements := make([]types.String, len(result.ScheduledQueries))
		for i, query := range result.ScheduledQueries {
			elements[i] = types.StringValue(query)
		}
		queriesList, diags := types.ListValueFrom(ctx, types.StringType, elements)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ScheduledQueries = queriesList
	} else {
		data.ScheduledQueries = types.ListNull(types.StringType)
	}

	// Set computed fields
	data.CreatedBy = resource_scheduled_rule.NewCreatedByValueNull()
	data.CreatedByExternal = types.StringNull()
	data.Managed = types.BoolNull()
	data.OutputIds = types.ListNull(types.StringType)
	data.Reports = types.MapNull(types.ListType{ElemType: types.StringType})
	data.SummaryAttributes = types.ListNull(types.StringType)
	data.Tests = types.ListNull(resource_scheduled_rule.TestsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_scheduled_rule.TestsValue{}.AttributeTypes(ctx),
		},
	})

	tflog.Debug(ctx, "Updated ScheduledRule", map[string]any{
		"id": result.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *scheduledRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_scheduled_rule.ScheduledRuleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteScheduledRule(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete scheduled_rule, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleted ScheduledRule", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *scheduledRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
