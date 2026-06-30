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
	"terraform-provider-panther/internal/provider/resource_logforwardersource"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const logForwarderSourcePath = "/log-sources/log-forwarder"

var (
	_ resource.Resource                = (*logforwardersourceResource)(nil)
	_ resource.ResourceWithConfigure   = (*logforwardersourceResource)(nil)
	_ resource.ResourceWithImportState = (*logforwardersourceResource)(nil)
)

func NewLogforwardersourceResource() resource.Resource {
	return &logforwardersourceResource{}
}

type logforwardersourceResource struct {
	rest *client.RESTClient
}

func (r *logforwardersourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logforwardersource"
}

func (r *logforwardersourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_logforwardersource.LogforwardersourceResourceSchema(ctx)
	resp.Schema.MarkdownDescription = "Represents a Panther Log Forwarder (PLF) Log Source in Panther"
	applySchemaOverrides(&resp.Schema, []SchemaOverride{
		{Name: "id", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
	})

	// log_stream_type_options: default "" for inner fields, null object default for the block itself
	logStreamTypeOptions := resp.Schema.Attributes["log_stream_type_options"].(schema.SingleNestedAttribute)

	jsonArrayEnvelopeField := logStreamTypeOptions.Attributes["json_array_envelope_field"].(schema.StringAttribute)
	jsonArrayEnvelopeField.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["json_array_envelope_field"] = jsonArrayEnvelopeField

	xmlRootElement := logStreamTypeOptions.Attributes["xml_root_element"].(schema.StringAttribute)
	xmlRootElement.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["xml_root_element"] = xmlRootElement

	logStreamTypeOptions.Default = objectdefault.StaticValue(types.ObjectNull(
		resource_logforwardersource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx),
	))

	resp.Schema.Attributes["log_stream_type_options"] = logStreamTypeOptions
}

func (r *logforwardersourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *logforwardersourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_logforwardersource.LogforwardersourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.LogForwarderSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogTypes:             listToStringSlice(ctx, data.LogTypes, &resp.Diagnostics),
		LogStreamTypeOptions: logForwarderLogStreamTypeOptions(data.LogStreamTypeOptions),
	}

	logForwarderSource, err := client.RestDo[client.LogForwarderSource](ctx, r.rest, http.MethodPost, logForwarderSourcePath, input)
	if handleCreateError(resp, "Log Forwarder Source", err) {
		return
	}
	tflog.Debug(ctx, "Created Log Forwarder Source", map[string]any{
		"id": logForwarderSource.IntegrationId,
	})
	data.Id = types.StringValue(logForwarderSource.IntegrationId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logforwardersourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_logforwardersource.LogforwardersourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	logForwarderSource, err := client.RestDo[client.LogForwarderSource](ctx, r.rest, http.MethodGet, logForwarderSourcePath+"/"+data.Id.ValueString(), nil)
	if handleReadError(ctx, resp, "Log Forwarder Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Got Log Forwarder Source", map[string]any{
		"id": logForwarderSource.IntegrationId,
	})

	data.Id = types.StringValue(logForwarderSource.IntegrationId)
	data.IntegrationLabel = types.StringValue(logForwarderSource.IntegrationLabel)
	data.LogStreamType = types.StringValue(logForwarderSource.LogStreamType)
	data.LogTypes = stringSliceToList(ctx, logForwarderSource.LogTypes, &resp.Diagnostics)

	if logForwarderSource.LogStreamTypeOptions != nil {
		attributeTypes := resource_logforwardersource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx)
		attributeValues := map[string]attr.Value{
			"json_array_envelope_field": types.StringValue(logForwarderSource.LogStreamTypeOptions.JsonArrayEnvelopeField),
			"xml_root_element":          types.StringValue(logForwarderSource.LogStreamTypeOptions.XmlRootElement),
		}

		logStreamTypeOptionsValue, diags := resource_logforwardersource.NewLogStreamTypeOptionsValue(attributeTypes, attributeValues)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
		} else {
			data.LogStreamTypeOptions = logStreamTypeOptionsValue
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logforwardersourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_logforwardersource.LogforwardersourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.LogForwarderSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogTypes:             listToStringSlice(ctx, data.LogTypes, &resp.Diagnostics),
		LogStreamTypeOptions: logForwarderLogStreamTypeOptions(data.LogStreamTypeOptions),
	}

	_, err := client.RestDo[client.LogForwarderSource](ctx, r.rest, http.MethodPut, logForwarderSourcePath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(ctx, resp, "Log Forwarder Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated Log Forwarder Source", map[string]any{
		"id": data.Id.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logforwardersourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_logforwardersource.LogforwardersourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, logForwarderSourcePath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "Log Forwarder Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Deleted Log Forwarder Source", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *logforwardersourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func logForwarderLogStreamTypeOptions(opts resource_logforwardersource.LogStreamTypeOptionsValue) *client.LogForwarderLogStreamTypeOptions {
	if opts.IsNull() {
		return nil
	}
	return &client.LogForwarderLogStreamTypeOptions{
		JsonArrayEnvelopeField: opts.JsonArrayEnvelopeField.ValueString(),
		XmlRootElement:         opts.XmlRootElement.ValueString(),
	}
}
