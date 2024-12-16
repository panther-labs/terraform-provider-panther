package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"terraform-provider-panther/internal/client"
	"terraform-provider-panther/internal/client/panther"
	"terraform-provider-panther/internal/provider/resource_httpsource"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = (*httpsourceResource)(nil)
	_ resource.ResourceWithConfigure = (*httpsourceResource)(nil)
)

func NewHttpsourceResource() resource.Resource {
	return &httpsourceResource{}
}

type httpsourceResource struct {
	client client.RestClient
}

func (r *httpsourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_httpsource"
}

func (r *httpsourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// We are overriding the schema here with some settings that are not supported by the schema generator.
	// We opt to do it here in order to be able to keep generating it without our changes getting overwritten in the generated file
	resp.Schema = resource_httpsource.HttpsourceResourceSchema(ctx)
	// we add the UseStateForUnknown plan modifier to the id attribute manually because it is not supported by the schema generator
	idAttr := resp.Schema.Attributes["id"].(schema.StringAttribute)
	idAttr.PlanModifiers = append(idAttr.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	resp.Schema.Attributes["id"] = idAttr

	// override default value for optional values
	secAlg := resp.Schema.Attributes["security_alg"].(schema.StringAttribute)
	secAlg.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["security_alg"] = secAlg

	secHeadKey := resp.Schema.Attributes["security_header_key"].(schema.StringAttribute)
	secHeadKey.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["security_header_key"] = secHeadKey

	secPass := resp.Schema.Attributes["security_password"].(schema.StringAttribute)
	secPass.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["security_password"] = secPass

	secSecVal := resp.Schema.Attributes["security_secret_value"].(schema.StringAttribute)
	secSecVal.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["security_secret_value"] = secSecVal

	secUser := resp.Schema.Attributes["security_username"].(schema.StringAttribute)
	secUser.Default = stringdefault.StaticString("")
	resp.Schema.Attributes["security_username"] = secUser
}

func (r *httpsourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

	// todo add nil check

	r.client = c.RestClient
}

func (r *httpsourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_httpsource.HttpsourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpSource, err := r.client.CreateHttpSource(ctx, client.CreateHttpSourceInput{
		HttpSourceModifiableAttributes: client.HttpSourceModifiableAttributes{
			IntegrationLabel:    data.IntegrationLabel.ValueString(),
			LogStreamType:       data.LogStreamType.ValueString(),
			LogTypes:            convertLogTypes(ctx, data.LogTypes),
			SecurityAlg:         data.SecurityAlg.ValueString(),
			SecurityHeaderKey:   data.SecurityHeaderKey.ValueString(),
			SecurityPassword:    data.SecurityPassword.ValueString(),
			SecuritySecretValue: data.SecuritySecretValue.ValueString(),
			SecurityType:        data.SecurityType.ValueString(),
			SecurityUsername:    data.SecurityUsername.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating HTTP Source",
			"Could not create HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}
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

	httpSource, err := r.client.GetHttpSource(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading HTTP Source",
			"Could not read HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}
	// We need to set all the values from the API response into the data model, except for the sensitive values
	// which are returned always as empty strings
	data.Id = types.StringValue(httpSource.IntegrationId)
	data.IntegrationLabel = types.StringValue(httpSource.IntegrationLabel)
	data.LogStreamType = types.StringValue(httpSource.LogStreamType)
	data.LogTypes = convertFromLogTypes(ctx, httpSource.LogTypes, resp.Diagnostics)
	data.SecurityType = types.StringValue(httpSource.SecurityType)
	data.SecurityAlg = types.StringValue(httpSource.SecurityAlg)
	data.SecurityHeaderKey = types.StringValue(httpSource.SecurityHeaderKey)
	data.SecurityUsername = types.StringValue(httpSource.SecurityUsername)

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

	_, err := r.client.UpdateHttpSource(ctx, client.UpdateHttpSourceInput{
		Id: data.Id.ValueString(),
		HttpSourceModifiableAttributes: client.HttpSourceModifiableAttributes{
			IntegrationLabel:    data.IntegrationLabel.ValueString(),
			LogStreamType:       data.LogStreamType.ValueString(),
			LogTypes:            convertLogTypes(ctx, data.LogTypes),
			SecurityAlg:         data.SecurityAlg.ValueString(),
			SecurityHeaderKey:   data.SecurityHeaderKey.ValueString(),
			SecurityPassword:    data.SecurityPassword.ValueString(),
			SecuritySecretValue: data.SecuritySecretValue.ValueString(),
			SecurityType:        data.SecurityType.ValueString(),
			SecurityUsername:    data.SecurityUsername.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating HTTP Source",
			"Could not update HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}

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

	err := r.client.DeleteHttpSource(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting HTTP Source",
			"Could not delete HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *httpsourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func convertLogTypes(ctx context.Context, logTypes types.List) []string {
	var result []string
	logTypes.ElementsAs(ctx, &result, false)
	return result
}

func convertFromLogTypes(ctx context.Context, logTypes []string, diagnostics diag.Diagnostics) types.List {
	from, d := types.ListValueFrom(ctx, types.StringType, logTypes)
	diagnostics.Append(d...)
	return from
}

func initialiseUnknownValues(r resource_httpsource.HttpsourceModel) resource_httpsource.HttpsourceModel {
	if r.SecurityAlg.IsUnknown() {
		r.SecurityAlg = types.StringValue("")
	}
	if r.SecurityHeaderKey.IsUnknown() {
		r.SecurityHeaderKey = types.StringValue("")
	}
	if r.SecuritySecretValue.IsUnknown() {
		r.SecuritySecretValue = types.StringValue("")
	}
	if r.SecurityUsername.IsUnknown() {
		r.SecurityUsername = types.StringValue("")
	}
	if r.SecurityPassword.IsUnknown() {
		r.SecurityPassword = types.StringValue("")
	}
	return r
}
