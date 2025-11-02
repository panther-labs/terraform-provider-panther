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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = (*CloudAccountResource)(nil)
	_ resource.ResourceWithImportState = (*CloudAccountResource)(nil)
	_ resource.ResourceWithConfigure   = (*CloudAccountResource)(nil)
)

func NewCloudAccountResource() resource.Resource {
	return &CloudAccountResource{}
}

type CloudAccountResource struct {
	client client.GraphQLClient
}

// CloudAccountResourceModel describes the resource data model.
type CloudAccountResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	AWSAccountID              types.String `tfsdk:"aws_account_id"`
	Label                     types.String `tfsdk:"label"`
	AuditRole                 types.String `tfsdk:"audit_role"`
	AWSStackName              types.String `tfsdk:"aws_stack_name"`
	AWSRegionIgnoreList       types.List   `tfsdk:"aws_region_ignore_list"`
	ResourceRegexIgnoreList   types.List   `tfsdk:"resource_regex_ignore_list"`
	ResourceTypeIgnoreList    types.List   `tfsdk:"resource_type_ignore_list"`
}

func (r *CloudAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_account"
}

func (r *CloudAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a Cloud Account integration in Panther",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the cloud account integration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"aws_account_id": schema.StringAttribute{
				Description:   "The AWS Account ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"label": schema.StringAttribute{
				Description: "The display name for this cloud account integration.",
				Required:    true,
			},
			"audit_role": schema.StringAttribute{
				Description: "The AWS IAM role ARN used for auditing resources.",
				Required:    true,
			},
			"aws_stack_name": schema.StringAttribute{
				Description: "The CloudFormation stack name (computed).",
				Computed:    true,
			},
			"aws_region_ignore_list": schema.ListAttribute{
				Description: "List of AWS regions to ignore during scanning.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"resource_regex_ignore_list": schema.ListAttribute{
				Description: "List of regex patterns for resource names to ignore during scanning.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"resource_type_ignore_list": schema.ListAttribute{
				Description: "List of AWS resource types to ignore during scanning.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *CloudAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*panther.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *panther.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client.GraphQLClient
}

// Helper function to convert types.List to []string
func listToStringSlice(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return []string{}
	}
	
	var elements []types.String
	list.ElementsAs(ctx, &elements, false)
	
	result := make([]string, len(elements))
	for i, elem := range elements {
		result[i] = elem.ValueString()
	}
	return result
}

// Helper function to convert []string to types.List
func stringSliceToList(ctx context.Context, slice []string) types.List {
	if slice == nil {
		return types.ListNull(types.StringType)
	}
	
	elements := make([]attr.Value, len(slice))
	for i, s := range slice {
		elements[i] = types.StringValue(s)
	}
	return types.ListValueMust(types.StringType, elements)
}

func (r *CloudAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CloudAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating cloud account", map[string]interface{}{
		"aws_account_id": data.AWSAccountID.ValueString(),
		"label":          data.Label.ValueString(),
		"audit_role":     data.AuditRole.ValueString(),
	})

	input := client.CreateCloudAccountInput{
		AWSAccountID: data.AWSAccountID.ValueString(),
		Label:        data.Label.ValueString(),
		AWSScanConfig: client.AWSScanConfigInput{
			AuditRole: data.AuditRole.ValueString(),
		},
		AWSRegionIgnoreList:     listToStringSlice(ctx, data.AWSRegionIgnoreList),
		ResourceRegexIgnoreList: listToStringSlice(ctx, data.ResourceRegexIgnoreList),
		ResourceTypeIgnoreList:  listToStringSlice(ctx, data.ResourceTypeIgnoreList),
	}

	result, err := r.client.CreateCloudAccount(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cloud account, got error: %s", err))
		return
	}

	if result.CloudAccount == nil {
		resp.Diagnostics.AddError("API Error", "CreateCloudAccount returned nil cloud account")
		return
	}

	// Map response to model
	data.ID = types.StringValue(result.CloudAccount.ID)
	data.AWSAccountID = types.StringValue(result.CloudAccount.AWSAccountID)
	data.Label = types.StringValue(result.CloudAccount.Label)
	data.AWSStackName = types.StringValue(result.CloudAccount.AWSStackName)
	data.AuditRole = types.StringValue(result.CloudAccount.AWSScanConfig.AuditRole)
	
	// Preserve null values from plan if they were originally null
	if data.AWSRegionIgnoreList.IsNull() && len(result.CloudAccount.AWSRegionIgnoreList) == 0 {
		data.AWSRegionIgnoreList = types.ListNull(types.StringType)
	} else {
		data.AWSRegionIgnoreList = stringSliceToList(ctx, result.CloudAccount.AWSRegionIgnoreList)
	}
	
	if data.ResourceRegexIgnoreList.IsNull() && len(result.CloudAccount.ResourceRegexIgnoreList) == 0 {
		data.ResourceRegexIgnoreList = types.ListNull(types.StringType)
	} else {
		data.ResourceRegexIgnoreList = stringSliceToList(ctx, result.CloudAccount.ResourceRegexIgnoreList)
	}
	
	if data.ResourceTypeIgnoreList.IsNull() && len(result.CloudAccount.ResourceTypeIgnoreList) == 0 {
		data.ResourceTypeIgnoreList = types.ListNull(types.StringType)
	} else {
		data.ResourceTypeIgnoreList = stringSliceToList(ctx, result.CloudAccount.ResourceTypeIgnoreList)
	}

	tflog.Debug(ctx, "Created cloud account", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CloudAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CloudAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading cloud account", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	cloudAccount, err := r.client.GetCloudAccount(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cloud account, got error: %s", err))
		return
	}

	if cloudAccount == nil {
		resp.Diagnostics.AddError("API Error", "GetCloudAccount returned nil cloud account")
		return
	}

	// Map response to model
	data.ID = types.StringValue(cloudAccount.ID)
	data.AWSAccountID = types.StringValue(cloudAccount.AWSAccountID)
	data.Label = types.StringValue(cloudAccount.Label)
	data.AWSStackName = types.StringValue(cloudAccount.AWSStackName)
	data.AuditRole = types.StringValue(cloudAccount.AWSScanConfig.AuditRole)
	
	// Preserve null values from state if they were originally null
	if data.AWSRegionIgnoreList.IsNull() && len(cloudAccount.AWSRegionIgnoreList) == 0 {
		data.AWSRegionIgnoreList = types.ListNull(types.StringType)
	} else {
		data.AWSRegionIgnoreList = stringSliceToList(ctx, cloudAccount.AWSRegionIgnoreList)
	}
	
	if data.ResourceRegexIgnoreList.IsNull() && len(cloudAccount.ResourceRegexIgnoreList) == 0 {
		data.ResourceRegexIgnoreList = types.ListNull(types.StringType)
	} else {
		data.ResourceRegexIgnoreList = stringSliceToList(ctx, cloudAccount.ResourceRegexIgnoreList)
	}
	
	if data.ResourceTypeIgnoreList.IsNull() && len(cloudAccount.ResourceTypeIgnoreList) == 0 {
		data.ResourceTypeIgnoreList = types.ListNull(types.StringType)
	} else {
		data.ResourceTypeIgnoreList = stringSliceToList(ctx, cloudAccount.ResourceTypeIgnoreList)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CloudAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CloudAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating cloud account", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	input := client.UpdateCloudAccountInput{
		ID:    data.ID.ValueString(),
		Label: data.Label.ValueString(),
		AWSScanConfig: client.AWSScanConfigInput{
			AuditRole: data.AuditRole.ValueString(),
		},
		AWSRegionIgnoreList:     listToStringSlice(ctx, data.AWSRegionIgnoreList),
		ResourceRegexIgnoreList: listToStringSlice(ctx, data.ResourceRegexIgnoreList),
		ResourceTypeIgnoreList:  listToStringSlice(ctx, data.ResourceTypeIgnoreList),
	}

	result, err := r.client.UpdateCloudAccount(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cloud account, got error: %s", err))
		return
	}

	if result.CloudAccount == nil {
		resp.Diagnostics.AddError("API Error", "UpdateCloudAccount returned nil cloud account")
		return
	}

	// Map response to model
	data.ID = types.StringValue(result.CloudAccount.ID)
	data.AWSAccountID = types.StringValue(result.CloudAccount.AWSAccountID)
	data.Label = types.StringValue(result.CloudAccount.Label)
	data.AWSStackName = types.StringValue(result.CloudAccount.AWSStackName)
	data.AuditRole = types.StringValue(result.CloudAccount.AWSScanConfig.AuditRole)
	
	// Preserve null values from plan if they were originally null
	if data.AWSRegionIgnoreList.IsNull() && len(result.CloudAccount.AWSRegionIgnoreList) == 0 {
		data.AWSRegionIgnoreList = types.ListNull(types.StringType)
	} else {
		data.AWSRegionIgnoreList = stringSliceToList(ctx, result.CloudAccount.AWSRegionIgnoreList)
	}
	
	if data.ResourceRegexIgnoreList.IsNull() && len(result.CloudAccount.ResourceRegexIgnoreList) == 0 {
		data.ResourceRegexIgnoreList = types.ListNull(types.StringType)
	} else {
		data.ResourceRegexIgnoreList = stringSliceToList(ctx, result.CloudAccount.ResourceRegexIgnoreList)
	}
	
	if data.ResourceTypeIgnoreList.IsNull() && len(result.CloudAccount.ResourceTypeIgnoreList) == 0 {
		data.ResourceTypeIgnoreList = types.ListNull(types.StringType)
	} else {
		data.ResourceTypeIgnoreList = stringSliceToList(ctx, result.CloudAccount.ResourceTypeIgnoreList)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CloudAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CloudAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting cloud account", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	input := client.DeleteCloudAccountInput{
		ID: data.ID.ValueString(),
	}

	_, err := r.client.DeleteCloudAccount(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cloud account, got error: %s", err))
		return
	}
}

func (r *CloudAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}