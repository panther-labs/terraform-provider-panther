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
	"terraform-provider-panther/internal/provider/resource_pubsubsource"

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

const pubsubSourcePath = "/log-sources/pubsub"

var (
	_ resource.Resource                = (*pubsubsourceResource)(nil)
	_ resource.ResourceWithConfigure   = (*pubsubsourceResource)(nil)
	_ resource.ResourceWithImportState = (*pubsubsourceResource)(nil)
)

func NewPubsubsourceResource() resource.Resource {
	return &pubsubsourceResource{}
}

type pubsubsourceResource struct {
	rest *client.RESTClient
}

func (r *pubsubsourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pubsubsource"
}

func (r *pubsubsourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_pubsubsource.PubsubsourceResourceSchema(ctx)
	patchIDAttribute(&resp.Schema)

	// Override Optional+Computed string attributes that the generator can't fully configure
	applySchemaOverrides(&resp.Schema, []SchemaOverride{
		{Name: "credentials", Default: stringdefault.StaticString(""), Sensitive: true},
		{Name: "project_id", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		{Name: "regional_endpoint", Default: stringdefault.StaticString("")},
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
		resource_pubsubsource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx),
	))

	resp.Schema.Attributes["log_stream_type_options"] = logStreamTypeOptions
}

func (r *pubsubsourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c := providerClients(req, resp)
	if c == nil {
		return
	}
	r.rest = c.REST
}

func (r *pubsubsourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_pubsubsource.PubsubsourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.PubSubSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		SubscriptionId:       data.SubscriptionId.ValueString(),
		ProjectId:            data.ProjectId.ValueString(),
		Credentials:          data.Credentials.ValueString(),
		CredentialsType:      data.CredentialsType.ValueString(),
		LogTypes:             convertLogTypes(ctx, data.LogTypes, &resp.Diagnostics),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogStreamTypeOptions: pubsubLogStreamTypeOptions(data.LogStreamTypeOptions),
		RegionalEndpoint:     data.RegionalEndpoint.ValueString(),
	}

	pubsubSource, err := client.RestDo[client.PubSubSource](ctx, r.rest, http.MethodPost, pubsubSourcePath, input)
	if handleCreateError(resp, "Pub/Sub Source", err) {
		return
	}
	tflog.Debug(ctx, "Created Pub/Sub Source", map[string]any{
		"id": pubsubSource.IntegrationId,
	})

	// Set server-assigned/derived fields from the API response
	data.Id = types.StringValue(pubsubSource.IntegrationId)
	// project_id: if unknown or null in the plan (user omitted it), resolve from the API response.
	// If the user provided a value, keep the plan value — Terraform rejects plan→apply changes.
	if data.ProjectId.IsUnknown() || data.ProjectId.IsNull() {
		data.ProjectId = types.StringValue(pubsubSource.ProjectId)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pubsubsourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_pubsubsource.PubsubsourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pubsubSource, err := client.RestDo[client.PubSubSource](ctx, r.rest, http.MethodGet, pubsubSourcePath+"/"+data.Id.ValueString(), nil)
	if handleReadError(ctx, resp, "Pub/Sub Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Got Pub/Sub Source", map[string]any{
		"id": pubsubSource.IntegrationId,
	})

	// Map all API response fields to state EXCEPT credentials.
	// The API always returns "" for credentials (sensitive/write-only).
	// The prior state value (from req.State.Get above) is preserved.
	data.Id = types.StringValue(pubsubSource.IntegrationId)
	data.IntegrationLabel = types.StringValue(pubsubSource.IntegrationLabel)
	data.SubscriptionId = types.StringValue(pubsubSource.SubscriptionId)
	data.ProjectId = types.StringValue(pubsubSource.ProjectId)
	data.CredentialsType = types.StringValue(pubsubSource.CredentialsType)
	data.LogTypes = convertFromLogTypes(ctx, pubsubSource.LogTypes, &resp.Diagnostics)
	data.LogStreamType = types.StringValue(pubsubSource.LogStreamType)
	data.RegionalEndpoint = types.StringValue(pubsubSource.RegionalEndpoint)

	if pubsubSource.LogStreamTypeOptions != nil {
		attributeTypes := resource_pubsubsource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx)
		attributeValues := map[string]attr.Value{
			"json_array_envelope_field": types.StringValue(pubsubSource.LogStreamTypeOptions.JsonArrayEnvelopeField),
			"xml_root_element":          types.StringValue(pubsubSource.LogStreamTypeOptions.XmlRootElement),
		}
		logStreamTypeOptionsValue, diags := resource_pubsubsource.NewLogStreamTypeOptionsValue(attributeTypes, attributeValues)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
		} else {
			data.LogStreamTypeOptions = logStreamTypeOptionsValue
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pubsubsourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_pubsubsource.PubsubsourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.PubSubSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		SubscriptionId:       data.SubscriptionId.ValueString(),
		ProjectId:            data.ProjectId.ValueString(),
		Credentials:          data.Credentials.ValueString(),
		CredentialsType:      data.CredentialsType.ValueString(),
		LogTypes:             convertLogTypes(ctx, data.LogTypes, &resp.Diagnostics),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogStreamTypeOptions: pubsubLogStreamTypeOptions(data.LogStreamTypeOptions),
		RegionalEndpoint:     data.RegionalEndpoint.ValueString(),
	}

	_, err := client.RestDo[client.PubSubSource](ctx, r.rest, http.MethodPut, pubsubSourcePath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(resp, "Pub/Sub Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated Pub/Sub Source", map[string]any{
		"id": data.Id.ValueString(),
	})

	// Save plan data to state (not full API response — credentials would be lost)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pubsubsourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_pubsubsource.PubsubsourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, pubsubSourcePath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "Pub/Sub Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Deleted Pub/Sub Source", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *pubsubsourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func pubsubLogStreamTypeOptions(opts resource_pubsubsource.LogStreamTypeOptionsValue) *client.PubSubLogStreamTypeOptions {
	if opts.IsNull() {
		return nil
	}
	return &client.PubSubLogStreamTypeOptions{
		JsonArrayEnvelopeField: opts.JsonArrayEnvelopeField.ValueString(),
		XmlRootElement:         opts.XmlRootElement.ValueString(),
	}
}
