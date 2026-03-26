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
	"terraform-provider-panther/internal/provider/resource_httpsource"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const httpSourcePath = "/log-sources/http"

var (
	_ resource.Resource                = (*httpsourceResource)(nil)
	_ resource.ResourceWithConfigure   = (*httpsourceResource)(nil)
	_ resource.ResourceWithImportState = (*httpsourceResource)(nil)
)

func NewHttpsourceResource() resource.Resource {
	return &httpsourceResource{}
}

type httpsourceResource struct {
	rest *client.RESTClient
}

func (r *httpsourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_httpsource"
}

func (r *httpsourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_httpsource.HttpsourceResourceSchema(ctx)
	patchIDAttribute(&resp.Schema)

	// Override Optional+Computed string attributes that the generator can't fully configure
	applySchemaOverrides(&resp.Schema, []SchemaOverride{
		{Name: "auth_hmac_alg", Default: stringdefault.StaticString("")},
		{Name: "auth_header_key", Default: stringdefault.StaticString("")},
		{Name: "auth_password", Default: stringdefault.StaticString(""), Sensitive: true},
		{Name: "auth_secret_value", Default: stringdefault.StaticString(""), Sensitive: true},
		{Name: "auth_username", Default: stringdefault.StaticString("")},
		{Name: "auth_bearer_token", Default: stringdefault.StaticString(""), Sensitive: true},
	})

	// logStreamTypeOptions: nested object needs inner defaults + null object default
	logStreamTypeOptions := resp.Schema.Attributes["log_stream_type_options"].(schema.SingleNestedAttribute)

	jsonArrayEnvelopeField := logStreamTypeOptions.Attributes["json_array_envelope_field"].(schema.StringAttribute)
	jsonArrayEnvelopeField.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["json_array_envelope_field"] = jsonArrayEnvelopeField

	xmlRootElement := logStreamTypeOptions.Attributes["xml_root_element"].(schema.StringAttribute)
	xmlRootElement.Default = stringdefault.StaticString("")
	logStreamTypeOptions.Attributes["xml_root_element"] = xmlRootElement

	logStreamTypeOptions.Default = objectdefault.StaticValue(types.ObjectNull(
		resource_httpsource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx),
	))

	resp.Schema.Attributes["log_stream_type_options"] = logStreamTypeOptions
}

func (r *httpsourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c := providerClients(req, resp)
	if c == nil {
		return
	}
	r.rest = c.REST
}

func (r *httpsourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_httpsource.HttpsourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.HttpSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogTypes:             convertLogTypes(ctx, data.LogTypes, &resp.Diagnostics),
		LogStreamTypeOptions: httpLogStreamTypeOptions(data.LogStreamTypeOptions),
		AuthHmacAlg:          data.AuthHmacAlg.ValueString(),
		AuthHeaderKey:        data.AuthHeaderKey.ValueString(),
		AuthPassword:         data.AuthPassword.ValueString(),
		AuthSecretValue:      data.AuthSecretValue.ValueString(),
		AuthMethod:           data.AuthMethod.ValueString(),
		AuthUsername:         data.AuthUsername.ValueString(),
		AuthBearerToken:      data.AuthBearerToken.ValueString(),
	}

	httpSource, err := client.RestDo[client.HttpSource](ctx, r.rest, http.MethodPost, httpSourcePath, input)
	if handleCreateError(resp, "HTTP Source", err) {
		return
	}
	tflog.Debug(ctx, "Created HTTP Source", map[string]any{
		"id": httpSource.IntegrationId,
	})
	data.Id = types.StringValue(httpSource.IntegrationId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpsourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_httpsource.HttpsourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpSource, err := client.RestDo[client.HttpSource](ctx, r.rest, http.MethodGet, httpSourcePath+"/"+data.Id.ValueString(), nil)
	if handleReadError(ctx, resp, "HTTP Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Got HTTP Source", map[string]any{
		"id": httpSource.IntegrationId,
	})
	// Sensitive fields (auth_password, auth_secret_value, auth_bearer_token) are returned as ""
	// by the API — don't overwrite state for those.
	data.Id = types.StringValue(httpSource.IntegrationId)
	data.IntegrationLabel = types.StringValue(httpSource.IntegrationLabel)
	data.LogStreamType = types.StringValue(httpSource.LogStreamType)
	data.LogTypes = convertFromLogTypes(ctx, httpSource.LogTypes, &resp.Diagnostics)
	data.AuthMethod = types.StringValue(httpSource.AuthMethod)
	data.AuthHmacAlg = types.StringValue(httpSource.AuthHmacAlg)
	data.AuthHeaderKey = types.StringValue(httpSource.AuthHeaderKey)
	data.AuthUsername = types.StringValue(httpSource.AuthUsername)

	if httpSource.LogStreamTypeOptions != nil {
		attributeTypes := resource_httpsource.LogStreamTypeOptionsValue{}.AttributeTypes(ctx)
		attributeValues := map[string]attr.Value{
			"json_array_envelope_field": types.StringValue(httpSource.LogStreamTypeOptions.JsonArrayEnvelopeField),
			"xml_root_element":          types.StringValue(httpSource.LogStreamTypeOptions.XmlRootElement),
		}

		logStreamTypeOptionsValue, diags := resource_httpsource.NewLogStreamTypeOptionsValue(attributeTypes, attributeValues)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
		} else {
			data.LogStreamTypeOptions = logStreamTypeOptionsValue
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpsourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_httpsource.HttpsourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.HttpSourceInput{
		IntegrationLabel:     data.IntegrationLabel.ValueString(),
		LogStreamType:        data.LogStreamType.ValueString(),
		LogTypes:             convertLogTypes(ctx, data.LogTypes, &resp.Diagnostics),
		LogStreamTypeOptions: httpLogStreamTypeOptions(data.LogStreamTypeOptions),
		AuthHmacAlg:          data.AuthHmacAlg.ValueString(),
		AuthHeaderKey:        data.AuthHeaderKey.ValueString(),
		AuthPassword:         data.AuthPassword.ValueString(),
		AuthSecretValue:      data.AuthSecretValue.ValueString(),
		AuthMethod:           data.AuthMethod.ValueString(),
		AuthUsername:         data.AuthUsername.ValueString(),
		AuthBearerToken:      data.AuthBearerToken.ValueString(),
	}

	_, err := client.RestDo[client.HttpSource](ctx, r.rest, http.MethodPut, httpSourcePath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(resp, "HTTP Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated HTTP Source", map[string]any{
		"id": data.Id.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpsourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_httpsource.HttpsourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, httpSourcePath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "HTTP Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Deleted HTTP Source", map[string]any{
		"id": data.Id.ValueString(),
	})
}

func (r *httpsourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func httpLogStreamTypeOptions(opts resource_httpsource.LogStreamTypeOptionsValue) *client.HttpLogStreamTypeOptions {
	if opts.IsNull() {
		return nil
	}
	return &client.HttpLogStreamTypeOptions{
		JsonArrayEnvelopeField: opts.JsonArrayEnvelopeField.ValueString(),
		XmlRootElement:         opts.XmlRootElement.ValueString(),
	}
}
