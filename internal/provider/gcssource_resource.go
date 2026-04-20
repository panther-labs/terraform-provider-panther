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
	"terraform-provider-panther/internal/provider/resource_gcssource"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const gcsSourcePath = "/log-sources/gcs"

var (
	_ resource.Resource                = (*gcssourceResource)(nil)
	_ resource.ResourceWithConfigure   = (*gcssourceResource)(nil)
	_ resource.ResourceWithImportState = (*gcssourceResource)(nil)
)

func NewGcssourceResource() resource.Resource {
	return &gcssourceResource{}
}

type gcssourceResource struct {
	rest *client.RESTClient
}

func (r *gcssourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gcssource"
}

func (r *gcssourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_gcssource.GcssourceResourceSchema(ctx)
	applySchemaOverrides(&resp.Schema, []SchemaOverride{
		{Name: "id", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		{Name: "credentials", Default: stringdefault.StaticString(""), Sensitive: true},
		{Name: "project_id", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
	})

	// log_stream_type_options: inner string field defaults + null object default
	logStreamTypeOptions := resp.Schema.Attributes["log_stream_type_options"].(schema.SingleNestedAttribute)

	jsonArrayEnvelopeField := logStreamTypeOptions.Attributes["json_array_envelope_field"].(schema.StringAttribute)
	jsonArrayEnvelopeField.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["json_array_envelope_field"] = jsonArrayEnvelopeField

	xmlRootElement := logStreamTypeOptions.Attributes["xml_root_element"].(schema.StringAttribute)
	xmlRootElement.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["xml_root_element"] = xmlRootElement

	logStreamTypeOptions.Default = objectdefault.StaticValue(types.ObjectNull(
		resource_gcssource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx),
	))

	resp.Schema.Attributes["log_stream_type_options"] = logStreamTypeOptions

	// prefix_log_types inner field overrides: prefix and excluded_prefixes need defaults
	prefixLogTypes := resp.Schema.Attributes["prefix_log_types"].(schema.ListNestedAttribute)

	prefix := prefixLogTypes.NestedObject.Attributes["prefix"].(schema.StringAttribute)
	prefix.Default = stringdefault.StaticString("")
	prefixLogTypes.NestedObject.Attributes["prefix"] = prefix

	excludedPrefixes := prefixLogTypes.NestedObject.Attributes["excluded_prefixes"].(schema.ListAttribute)
	excludedPrefixes.Default = listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{}))
	prefixLogTypes.NestedObject.Attributes["excluded_prefixes"] = excludedPrefixes

	logTypesAttr := prefixLogTypes.NestedObject.Attributes["log_types"].(schema.ListAttribute)
	logTypesAttr.Required = true
	logTypesAttr.Optional = false
	logTypesAttr.Computed = false
	prefixLogTypes.NestedObject.Attributes["log_types"] = logTypesAttr

	resp.Schema.Attributes["prefix_log_types"] = prefixLogTypes
}

func (r *gcssourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *gcssourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_gcssource.GcssourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.GcsSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		SubscriptionId:       data.SubscriptionId.ValueString(),
		ProjectId:            data.ProjectId.ValueString(),
		GcsBucket:            data.GcsBucket.ValueString(),
		Credentials:          data.Credentials.ValueString(),
		CredentialsType:      data.CredentialsType.ValueString(),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogStreamTypeOptions: gcsLogStreamTypeOptions(data.LogStreamTypeOptions),
		PrefixLogTypes:       gcsPrefixLogTypesToInput(ctx, data.PrefixLogTypes, &resp.Diagnostics),
	}

	gcsSource, err := client.RestDo[client.GcsSource](ctx, r.rest, http.MethodPost, gcsSourcePath, input)
	if handleCreateError(resp, "GCS Source", err) {
		return
	}
	tflog.Debug(ctx, "Created GCS Source", map[string]any{
		"id": gcsSource.IntegrationId,
	})

	data.Id = types.StringValue(gcsSource.IntegrationId)
	// project_id: if unknown or null in the plan (user omitted it), resolve from the API response.
	if data.ProjectId.IsUnknown() || data.ProjectId.IsNull() {
		data.ProjectId = types.StringValue(gcsSource.ProjectId)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gcssourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_gcssource.GcssourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gcsSource, err := client.RestDo[client.GcsSource](ctx, r.rest, http.MethodGet, gcsSourcePath+"/"+data.Id.ValueString(), nil)
	if handleReadError(ctx, resp, "GCS Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Got GCS Source", map[string]any{
		"id": gcsSource.IntegrationId,
	})

	// Map all API response fields to state EXCEPT credentials.
	// The API always returns "" for credentials (sensitive/write-only).
	data.Id = types.StringValue(gcsSource.IntegrationId)
	data.IntegrationLabel = types.StringValue(gcsSource.IntegrationLabel)
	data.SubscriptionId = types.StringValue(gcsSource.SubscriptionId)
	data.ProjectId = types.StringValue(gcsSource.ProjectId)
	data.GcsBucket = types.StringValue(gcsSource.GcsBucket)
	data.CredentialsType = types.StringValue(gcsSource.CredentialsType)
	data.LogStreamType = types.StringValue(gcsSource.LogStreamType)

	if gcsSource.LogStreamTypeOptions != nil {
		attributeTypes := resource_gcssource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx)
		attributeValues := map[string]attr.Value{
			"json_array_envelope_field": types.StringValue(gcsSource.LogStreamTypeOptions.JsonArrayEnvelopeField),
			"xml_root_element":          types.StringValue(gcsSource.LogStreamTypeOptions.XmlRootElement),
		}
		logStreamTypeOptionsValue, diags := resource_gcssource.NewLogStreamTypeOptionsValue(attributeTypes, attributeValues)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
		} else {
			data.LogStreamTypeOptions = logStreamTypeOptionsValue
		}
	}

	data.PrefixLogTypes = gcsPrefixLogTypesFromResponse(ctx, gcsSource.PrefixLogTypes, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gcssourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_gcssource.GcssourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.GcsSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		SubscriptionId:       data.SubscriptionId.ValueString(),
		ProjectId:            data.ProjectId.ValueString(),
		GcsBucket:            data.GcsBucket.ValueString(),
		Credentials:          data.Credentials.ValueString(),
		CredentialsType:      data.CredentialsType.ValueString(),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogStreamTypeOptions: gcsLogStreamTypeOptions(data.LogStreamTypeOptions),
		PrefixLogTypes:       gcsPrefixLogTypesToInput(ctx, data.PrefixLogTypes, &resp.Diagnostics),
	}

	_, err := client.RestDo[client.GcsSource](ctx, r.rest, http.MethodPut, gcsSourcePath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(resp, "GCS Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated GCS Source", map[string]any{
		"id": data.Id.ValueString(),
	})

	// Save plan data to state (not full API response — credentials would be lost)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gcssourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_gcssource.GcssourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, gcsSourcePath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "GCS Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Deleted GCS Source", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *gcssourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func gcsLogStreamTypeOptions(opts resource_gcssource.LogStreamTypeOptionsValue) *client.GcsLogStreamTypeOptions {
	if opts.IsNull() {
		return nil
	}
	return &client.GcsLogStreamTypeOptions{
		JsonArrayEnvelopeField: opts.JsonArrayEnvelopeField.ValueString(),
		XmlRootElement:         opts.XmlRootElement.ValueString(),
	}
}

// gcsPrefixLogTypesToInput converts the Terraform model list to client input structs.
func gcsPrefixLogTypesToInput(ctx context.Context, tfList types.List, diagnostics *diag.Diagnostics) []client.GcsPrefixLogTypesInput {
	var elements []resource_gcssource.PrefixLogTypesValue
	diagnostics.Append(tfList.ElementsAs(ctx, &elements, false)...)

	result := make([]client.GcsPrefixLogTypesInput, 0, len(elements))
	for _, e := range elements {
		var logTypes []string
		diagnostics.Append(e.LogTypes.ElementsAs(ctx, &logTypes, false)...)

		var excludedPrefixes []string
		diagnostics.Append(e.ExcludedPrefixes.ElementsAs(ctx, &excludedPrefixes, false)...)

		result = append(result, client.GcsPrefixLogTypesInput{
			Prefix:           e.Prefix.ValueString(),
			LogTypes:         logTypes,
			ExcludedPrefixes: excludedPrefixes,
		})
	}
	return result
}

// gcsPrefixLogTypesFromResponse converts API response prefix mappings to the Terraform model list.
func gcsPrefixLogTypesFromResponse(ctx context.Context, apiPrefixes []client.GcsPrefixLogTypesInput, diagnostics *diag.Diagnostics) types.List {
	attrTypes := resource_gcssource.PrefixLogTypesValue{}.AttributeTypes(ctx)
	elemType := resource_gcssource.PrefixLogTypesType{
		ObjectType: types.ObjectType{AttrTypes: attrTypes},
	}

	if len(apiPrefixes) == 0 {
		return types.ListValueMust(elemType, []attr.Value{})
	}

	elements := make([]attr.Value, 0, len(apiPrefixes))
	for _, p := range apiPrefixes {
		logTypes, d := types.ListValueFrom(ctx, types.StringType, p.LogTypes)
		diagnostics.Append(d...)

		excluded := p.ExcludedPrefixes
		if excluded == nil {
			excluded = []string{}
		}
		excludedPrefixes, d := types.ListValueFrom(ctx, types.StringType, excluded)
		diagnostics.Append(d...)

		val, d := resource_gcssource.NewPrefixLogTypesValue(attrTypes, map[string]attr.Value{
			"prefix":            types.StringValue(p.Prefix),
			"log_types":         logTypes,
			"excluded_prefixes": excludedPrefixes,
		})
		diagnostics.Append(d...)
		elements = append(elements, val)
	}

	list, d := types.ListValue(elemType, elements)
	diagnostics.Append(d...)
	return list
}
