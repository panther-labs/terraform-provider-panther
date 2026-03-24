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
	api *client.RESTResource[client.CreateHttpSourceInput, client.UpdateHttpSourceInput, client.HttpSource]
}

func (r *httpsourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_httpsource"
}

func (r *httpsourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// We are overriding the schema here with some settings that are not supported by the schema generator.
	// We opt to do it here in order to be able to keep generating it without our changes getting overwritten in the generated file
	resp.Schema = resource_httpsource.HttpsourceResourceSchema(ctx)
	patchIDAttribute(&resp.Schema)

	// override default value for optional values
	// this is necessary because the code generator creates these fields as optional and computed, which means that if they are not provided
	// by the user they will be unknown values.
	hmacAlg := resp.Schema.Attributes["auth_hmac_alg"].(schema.StringAttribute)
	hmacAlg.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["auth_hmac_alg"] = hmacAlg

	authHeadKey := resp.Schema.Attributes["auth_header_key"].(schema.StringAttribute)
	authHeadKey.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["auth_header_key"] = authHeadKey

	authPass := resp.Schema.Attributes["auth_password"].(schema.StringAttribute)
	authPass.Default = stringdefault.StaticString("")
	authPass.Sensitive = true
	resp.Schema.Attributes["auth_password"] = authPass

	authSecVal := resp.Schema.Attributes["auth_secret_value"].(schema.StringAttribute)
	authSecVal.Default = stringdefault.StaticString("")
	authSecVal.Sensitive = true
	resp.Schema.Attributes["auth_secret_value"] = authSecVal

	authUser := resp.Schema.Attributes["auth_username"].(schema.StringAttribute)
	authUser.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["auth_username"] = authUser

	bearerToken := resp.Schema.Attributes["auth_bearer_token"].(schema.StringAttribute)
	bearerToken.Default = stringdefault.StaticString("")
	bearerToken.Sensitive = true
	resp.Schema.Attributes["auth_bearer_token"] = bearerToken

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
	r.api = client.NewRESTResource[client.CreateHttpSourceInput, client.UpdateHttpSourceInput, client.HttpSource](
		c.REST, httpSourcePath,
		func(u client.UpdateHttpSourceInput) string { return u.IntegrationId },
	)
}

func (r *httpsourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_httpsource.HttpsourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateHttpSourceInput{
		HttpSourceModifiableAttributes: client.HttpSourceModifiableAttributes{
			IntegrationLabel: data.IntegrationLabel.ValueString(),
			LogStreamType:    data.LogStreamType.ValueString(),
			LogTypes:         convertLogTypes(ctx, data.LogTypes, resp.Diagnostics),
			AuthHmacAlg:      data.AuthHmacAlg.ValueString(),
			AuthHeaderKey:    data.AuthHeaderKey.ValueString(),
			AuthPassword:     data.AuthPassword.ValueString(),
			AuthSecretValue:  data.AuthSecretValue.ValueString(),
			AuthMethod:       data.AuthMethod.ValueString(),
			AuthUsername:     data.AuthUsername.ValueString(),
			AuthBearerToken:  data.AuthBearerToken.ValueString(),
		},
	}

	if !data.LogStreamTypeOptions.IsNull() {
		input.HttpSourceModifiableAttributes.LogStreamTypeOptions = &client.HttpLogStreamTypeOptions{
			JsonArrayEnvelopeField: data.LogStreamTypeOptions.JsonArrayEnvelopeField.ValueString(),
			XmlRootElement:         data.LogStreamTypeOptions.XmlRootElement.ValueString(),
		}
	}

	httpSource, err := r.api.Create(ctx, input)
	if handleCreateError(resp, "HTTP Source", err) {
		return
	}
	tflog.Debug(ctx, "Created HTTP Source", map[string]any{
		"id": httpSource.IntegrationId,
	})
	data.Id = types.StringValue(httpSource.IntegrationId)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpsourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_httpsource.HttpsourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpSource, err := r.api.Get(ctx, data.Id.ValueString())
	if handleReadError(ctx, resp, "HTTP Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Got HTTP Source", map[string]any{
		"id": httpSource.IntegrationId,
	})
	// We need to set all the values from the API response into the data model, except for the sensitive values
	// which are returned always as empty strings
	data.Id = types.StringValue(httpSource.IntegrationId)
	data.IntegrationLabel = types.StringValue(httpSource.IntegrationLabel)
	data.LogStreamType = types.StringValue(httpSource.LogStreamType)
	data.LogTypes = convertFromLogTypes(ctx, httpSource.LogTypes, resp.Diagnostics)
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
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpsourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_httpsource.HttpsourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateHttpSourceInput{
		IntegrationId: data.Id.ValueString(),
		HttpSourceModifiableAttributes: client.HttpSourceModifiableAttributes{
			IntegrationLabel: data.IntegrationLabel.ValueString(),
			LogStreamType:    data.LogStreamType.ValueString(),
			LogTypes:         convertLogTypes(ctx, data.LogTypes, resp.Diagnostics),
			AuthHmacAlg:      data.AuthHmacAlg.ValueString(),
			AuthHeaderKey:    data.AuthHeaderKey.ValueString(),
			AuthPassword:     data.AuthPassword.ValueString(),
			AuthSecretValue:  data.AuthSecretValue.ValueString(),
			AuthMethod:       data.AuthMethod.ValueString(),
			AuthUsername:     data.AuthUsername.ValueString(),
			AuthBearerToken:  data.AuthBearerToken.ValueString(),
		},
	}

	if !data.LogStreamTypeOptions.IsNull() {
		input.HttpSourceModifiableAttributes.LogStreamTypeOptions = &client.HttpLogStreamTypeOptions{
			JsonArrayEnvelopeField: data.LogStreamTypeOptions.JsonArrayEnvelopeField.ValueString(),
			XmlRootElement:         data.LogStreamTypeOptions.XmlRootElement.ValueString(),
		}
	}

	_, err := r.api.Update(ctx, input)
	if handleUpdateError(resp, "HTTP Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated HTTP Source", map[string]any{
		"id": data.Id.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpsourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_httpsource.HttpsourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.api.Delete(ctx, data.Id.ValueString())
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
