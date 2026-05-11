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
	"net/http"
	"strings"

	"terraform-provider-panther/internal/client"
	"terraform-provider-panther/internal/provider/resource_log_source_alarm"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const logSourceAlarmPath = "/log-source-alarms"

// AlarmTypeSourceNoData is the only alarm type the REST API currently exposes. The four
// system-managed types (permissions, classification, processing, scanning) are intentionally
// not surfaced (see panther-enterprise PR #28642).
const AlarmTypeSourceNoData = "SOURCE_NO_DATA"

var (
	_ resource.Resource                = (*logSourceAlarmResource)(nil)
	_ resource.ResourceWithConfigure   = (*logSourceAlarmResource)(nil)
	_ resource.ResourceWithImportState = (*logSourceAlarmResource)(nil)
)

func NewLogSourceAlarmResource() resource.Resource {
	return &logSourceAlarmResource{}
}

type logSourceAlarmResource struct {
	rest *client.RESTClient
}

// logSourceAlarmModel mirrors the generated resource_log_source_alarm.LogSourceAlarmModel
// plus the synthetic composite "id" attribute we layer on in Schema. Defined locally so
// the tfsdk tags match the augmented schema exactly; regenerating the package does not
// disturb this file.
type logSourceAlarmModel struct {
	Id               types.String `tfsdk:"id"`
	SourceId         types.String `tfsdk:"source_id"`
	Type             types.String `tfsdk:"type"`
	MinutesThreshold types.Int64  `tfsdk:"minutes_threshold"`
}

func (r *logSourceAlarmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_source_alarm"
}

func (r *logSourceAlarmResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_log_source_alarm.LogSourceAlarmResourceSchema(ctx)

	// Scope rationale (kept as a code comment, not user-facing docs): only the
	// SOURCE_NO_DATA alarm is configurable here. The four system-managed types
	// (SOURCE_PERMISSIONS_CHECKS, SOURCE_CLASSIFICATION_FAILURES,
	// SOURCE_LOG_PROCESSING_ERRORS, SOURCE_SCANNING_ERRORS) flip OK/ALARM at
	// runtime based on conditions the user doesn't directly control, so they
	// don't fit Terraform's declarative plan/apply model. The OneOf validator
	// on the `type` attribute enforces the same scope at the schema layer.
	resp.Schema.Description = "Manages a no-data drop-off alarm for a Panther log source"
	resp.Schema.MarkdownDescription = "Manages a no-data drop-off alarm for a Panther log source"

	// The generator models path parameters as computed/optional based on OpenAPI inference.
	// Both source_id and type are user-supplied path parameters; rewrite them as Required
	// with RequiresReplace (changing either identifies a different alarm resource).
	resp.Schema.Attributes["source_id"] = schema.StringAttribute{
		Required: true,
		Description: "The ID of the log source this alarm monitors (the `id` of a " +
			"`panther_s3_source`, `panther_httpsource`, `panther_gcssource`, or `panther_pubsubsource`). " +
			"Changing this forces resource recreation.",
		MarkdownDescription: "The ID of the log source this alarm monitors (the `id` of a " +
			"`panther_s3_source`, `panther_httpsource`, `panther_gcssource`, or `panther_pubsubsource`). " +
			"Changing this forces resource recreation.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	resp.Schema.Attributes["type"] = schema.StringAttribute{
		Required: true,
		Description: fmt.Sprintf(
			"The alarm type. Must be %q. Changing this forces resource recreation.",
			AlarmTypeSourceNoData,
		),
		MarkdownDescription: fmt.Sprintf(
			"The alarm type. Must be `%s`. Changing this forces resource recreation.",
			AlarmTypeSourceNoData,
		),
		Validators:    []validator.String{stringvalidator.OneOf(AlarmTypeSourceNoData)},
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}

	// Synthetic composite id so ImportStatePassthroughID and state-path conventions work.
	// The REST API has no single-scalar identifier — the URL path {sourceId}/{type} is the
	// natural unique key, and matches the Panther UI's canonical reference form.
	resp.Schema.Attributes["id"] = schema.StringAttribute{
		Computed:            true,
		Description:         `Composite identifier in the form "{source_id}/{type}".`,
		MarkdownDescription: "Composite identifier in the form `{source_id}/{type}`.",
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}

	// minutes_threshold: attach provider-side bounds. The OpenAPI spec intentionally omits
	// JSON-schema minimum/maximum (service-layer enforcement with customer-facing grammar),
	// so we wrap it here for fail-fast plan-time validation.
	mt := resp.Schema.Attributes["minutes_threshold"].(schema.Int64Attribute)
	mt.Validators = append(mt.Validators, int64validator.Between(15, 43200))
	resp.Schema.Attributes["minutes_threshold"] = mt
}

func (r *logSourceAlarmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *logSourceAlarmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data logSourceAlarmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqPath := alarmPath(data.SourceId.ValueString(), data.Type.ValueString())
	input := client.LogSourceAlarmInput{
		MinutesThreshold: data.MinutesThreshold.ValueInt64(),
	}
	putResp, err := client.RestDo[client.LogSourceAlarm](ctx, r.rest, http.MethodPut, reqPath, input)
	if handleCreateError(resp, "Log Source Alarm", err) {
		return
	}
	tflog.Debug(ctx, "Created Log Source Alarm", map[string]any{
		"source_id": data.SourceId.ValueString(),
		"type":      data.Type.ValueString(),
	})

	data.Id = types.StringValue(data.SourceId.ValueString() + "/" + data.Type.ValueString())
	data.Type = types.StringValue(putResp.Type)
	data.MinutesThreshold = types.Int64Value(putResp.MinutesThreshold)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logSourceAlarmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data logSourceAlarmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqPath := alarmPath(data.SourceId.ValueString(), data.Type.ValueString())
	alarm, err := client.RestDo[client.LogSourceAlarm](ctx, r.rest, http.MethodGet, reqPath, nil)
	if handleReadError(ctx, resp, "Log Source Alarm", data.Id.ValueString(), err) {
		return
	}

	data.MinutesThreshold = types.Int64Value(alarm.MinutesThreshold)
	data.Type = types.StringValue(alarm.Type)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logSourceAlarmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data logSourceAlarmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqPath := alarmPath(data.SourceId.ValueString(), data.Type.ValueString())
	input := client.LogSourceAlarmInput{
		MinutesThreshold: data.MinutesThreshold.ValueInt64(),
	}
	putResp, err := client.RestDo[client.LogSourceAlarm](ctx, r.rest, http.MethodPut, reqPath, input)
	if handleUpdateError(resp, "Log Source Alarm", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated Log Source Alarm", map[string]any{
		"id": data.Id.ValueString(),
	})

	data.Type = types.StringValue(putResp.Type)
	data.MinutesThreshold = types.Int64Value(putResp.MinutesThreshold)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logSourceAlarmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data logSourceAlarmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqPath := alarmPath(data.SourceId.ValueString(), data.Type.ValueString())
	err := client.RestDelete(ctx, r.rest, reqPath)
	if handleDeleteError(resp, "Log Source Alarm", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Deleted Log Source Alarm", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *logSourceAlarmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf(`Expected "{source_id}/{type}" (e.g. "41ed10a4-.../%s"), got: %q`, AlarmTypeSourceNoData, req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[1])...)
}

func alarmPath(sourceID, alarmType string) string {
	return logSourceAlarmPath + "/" + sourceID + "/" + alarmType
}
