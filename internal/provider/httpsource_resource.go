package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	resp.Schema = resource_httpsource.HttpsourceResourceSchema(ctx)
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
	// Create API call logic
	// fixme change inputs
	httpSource, err := r.client.CreateHttpSource(ctx, client.CreateHttpSourceInput{
		// fill all the fields from the data model
		IntegrationLabel:    data.IntegrationLabel.ValueString(),
		LogStreamType:       data.LogStreamType.ValueString(),
		LogTypes:            convertLogTypes(ctx, data.LogTypes),
		SecurityAlg:         data.SecurityAlg.ValueString(),
		SecurityHeaderKey:   data.SecurityHeaderKey.ValueString(),
		SecurityPassword:    data.SecurityPassword.ValueString(),
		SecuritySecretValue: data.SecuritySecretValue.ValueString(),
		SecurityType:        data.SecurityType.ValueString(),
		SecurityUsername:    data.SecurityUsername.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating HTTP Source",
			"Could not create HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}
	// Example data value setting
	data.Id = types.StringValue(httpSource.IntegrationId)
	data.IntegrationId = types.StringValue(httpSource.IntegrationId)

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

	// Read API call logic
	httpSource, err := r.client.GetHttpSource(ctx, data.IntegrationId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading HTTP Source",
			"Could not read HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}
	// Example data value setting
	data.Id = types.StringValue(httpSource.IntegrationId)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpsourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_httpsource.HttpsourceModel
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	tflog.Warn(ctx, "data: ", map[string]interface{}{
		"id": id.ValueString(),
	})

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	httpSource, err := r.client.UpdateHttpSource(ctx, client.UpdateHttpSourceInput{
		// fill all the fields from the data model
		Id:                  id.ValueString(),
		IntegrationLabel:    data.IntegrationLabel.ValueString(),
		LogStreamType:       data.LogStreamType.ValueString(),
		LogTypes:            convertLogTypes(ctx, data.LogTypes),
		SecurityAlg:         data.SecurityAlg.ValueString(),
		SecurityHeaderKey:   data.SecurityHeaderKey.ValueString(),
		SecurityPassword:    data.SecurityPassword.ValueString(),
		SecuritySecretValue: data.SecuritySecretValue.ValueString(),
		SecurityType:        data.SecurityType.ValueString(),
		SecurityUsername:    data.SecurityUsername.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating HTTP Source",
			"Could not update HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}
	// Example data value setting
	// fixme not there for s3
	data.IntegrationId = types.StringValue(httpSource.IntegrationId)
	data.Id = types.StringValue(httpSource.IntegrationId)

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

	// Delete API call logic
	err := r.client.DeleteHttpSource(ctx, data.IntegrationId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting HTTP Source",
			"Could not delete HTTP Source, unexpected error: "+err.Error(),
		)
		return
	}

}

func convertLogTypes(ctx context.Context, logTypes types.List) []string {
	var result []string
	logTypes.ElementsAs(ctx, &result, false)
	return result
}
