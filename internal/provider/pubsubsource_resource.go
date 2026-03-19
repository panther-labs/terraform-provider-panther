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
	"terraform-provider-panther/internal/provider/resource_pubsubsource"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = (*pubsubsourceResource)(nil)
	_ resource.ResourceWithConfigure = (*pubsubsourceResource)(nil)
)

func NewPubsubsourceResource() resource.Resource {
	return &pubsubsourceResource{}
}

type pubsubsourceResource struct {
	client client.RestClient
}

func (r *pubsubsourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pubsubsource"
}

func (r *pubsubsourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Start from the generated schema and apply overrides that the generator can't express
	resp.Schema = resource_pubsubsource.PubsubsourceResourceSchema(ctx)

	// id: UseStateForUnknown tells Terraform the server assigns the ID on create and it won't change
	idAttr := resp.Schema.Attributes["id"].(schema.StringAttribute)
	idAttr.PlanModifiers = append(idAttr.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	resp.Schema.Attributes["id"] = idAttr

	// credentials: sensitive (the API returns "" on read) + default "" to avoid unknown on omission
	credentials := resp.Schema.Attributes["credentials"].(schema.StringAttribute)
	credentials.Sensitive = true
	credentials.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["credentials"] = credentials

	// project_id: UseStateForUnknown — optional for SA (API derives from keyfile), required for WIF.
	// On create the API returns the derived value; on subsequent plans, prior state is reused.
	projectId := resp.Schema.Attributes["project_id"].(schema.StringAttribute)
	projectId.PlanModifiers = append(projectId.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	resp.Schema.Attributes["project_id"] = projectId

	// regional_endpoint: default "" to avoid unknown when not set
	regionalEndpoint := resp.Schema.Attributes["regional_endpoint"].(schema.StringAttribute)
	regionalEndpoint.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["regional_endpoint"] = regionalEndpoint

	// log_stream_type_options: default "" for inner fields, null object default for the block itself
	logStreamTypeOptions := resp.Schema.Attributes["log_stream_type_options"].(schema.SingleNestedAttribute)

	jsonArrayEnvelopeField := logStreamTypeOptions.Attributes["json_array_envelope_field"].(schema.StringAttribute)
	jsonArrayEnvelopeField.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["json_array_envelope_field"] = jsonArrayEnvelopeField

	xmlRootElement := logStreamTypeOptions.Attributes["xml_root_element"].(schema.StringAttribute)
	xmlRootElement.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["xml_root_element"] = xmlRootElement

	logStreamTypeOptions.Default = objectdefault.StaticValue(types.ObjectNull(
		map[string]attr.Type{
			"json_array_envelope_field": types.StringType,
			"xml_root_element":          types.StringType,
		},
	))

	resp.Schema.Attributes["log_stream_type_options"] = logStreamTypeOptions
}

func (r *pubsubsourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pubsubsourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_pubsubsource.PubsubsourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreatePubSubSourceInput{
		PubSubSourceModifiableAttributes: client.PubSubSourceModifiableAttributes{
			IntegrationLabel: data.IntegrationLabel.ValueString(),
			SubscriptionId:   data.SubscriptionId.ValueString(),
			ProjectId:        data.ProjectId.ValueString(),
			Credentials:      data.Credentials.ValueString(),
			CredentialsType:  data.CredentialsType.ValueString(),
			LogTypes:         convertLogTypes(ctx, data.LogTypes),
			LogStreamType:    data.LogStreamType.ValueString(),
			RegionalEndpoint: data.RegionalEndpoint.ValueString(),
		},
	}

	input.PubSubSourceModifiableAttributes.LogStreamTypeOptions = pubsubLogStreamTypeOptions(data.LogStreamTypeOptions)

	pubsubSource, err := r.client.CreatePubSubSource(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Pub/Sub Source",
			"Could not create Pub/Sub Source, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Created Pub/Sub Source", map[string]any{
		"id": pubsubSource.IntegrationId,
	})

	// Set server-assigned/derived fields from the API response
	data.Id = types.StringValue(pubsubSource.IntegrationId)
	// project_id may be derived from SA keyfile when omitted by the user
	data.ProjectId = types.StringValue(pubsubSource.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pubsubsourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_pubsubsource.PubsubsourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pubsubSource, err := r.client.GetPubSubSource(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Pub/Sub Source",
			fmt.Sprintf("Could not read Pub/Sub Source with id %s, unexpected error: %s", data.Id.ValueString(), err.Error()),
		)
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
	data.LogTypes = convertFromLogTypes(ctx, pubsubSource.LogTypes, resp.Diagnostics)
	data.LogStreamType = types.StringValue(pubsubSource.LogStreamType)
	data.RegionalEndpoint = types.StringValue(pubsubSource.RegionalEndpoint)

	if pubsubSource.LogStreamTypeOptions != nil {
		attributeTypes := map[string]attr.Type{
			"json_array_envelope_field": types.StringType,
			"xml_root_element":          types.StringType,
		}
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

	input := client.UpdatePubSubSourceInput{
		IntegrationId: data.Id.ValueString(),
		PubSubSourceModifiableAttributes: client.PubSubSourceModifiableAttributes{
			IntegrationLabel: data.IntegrationLabel.ValueString(),
			SubscriptionId:   data.SubscriptionId.ValueString(),
			ProjectId:        data.ProjectId.ValueString(),
			Credentials:      data.Credentials.ValueString(),
			CredentialsType:  data.CredentialsType.ValueString(),
			LogTypes:         convertLogTypes(ctx, data.LogTypes),
			LogStreamType:    data.LogStreamType.ValueString(),
			RegionalEndpoint: data.RegionalEndpoint.ValueString(),
		},
	}

	input.PubSubSourceModifiableAttributes.LogStreamTypeOptions = pubsubLogStreamTypeOptions(data.LogStreamTypeOptions)

	_, err := r.client.UpdatePubSubSource(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Pub/Sub Source",
			fmt.Sprintf("Could not update Pub/Sub Source with id %s, unexpected error: %s", data.Id.ValueString(), err.Error()),
		)
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

	err := r.client.DeletePubSubSource(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Pub/Sub Source",
			fmt.Sprintf("Could not delete Pub/Sub Source with id %s, unexpected error: %s", data.Id.ValueString(), err.Error()),
		)
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
