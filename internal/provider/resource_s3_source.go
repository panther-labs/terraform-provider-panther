// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-panther/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3SourceResource{}
	_ resource.ResourceWithImportState = &S3SourceResource{}
	_ resource.ResourceWithConfigure   = &S3SourceResource{}
)

func NewS3SourceResource() resource.Resource {
	return &S3SourceResource{}
}

type S3SourceResource struct {
	client client.Client
}

// ExampleResourceModel describes the resource data model.
type S3SourceResourceModel struct {
	AWSAccountID                        types.String          `tfsdk:"aws_account_id"`
	KMSKeyARN                           types.String          `tfsdk:"kms_key_arn"`
	Name                                types.String          `tfsdk:"name"`
	LogProcessingRoleARN                types.String          `tfsdk:"log_processing_role_arn"`
	LogStreamType                       types.String          `tfsdk:"log_stream_type"`
	IsManagedBucketNotificationsEnabled types.Bool            `tfsdk:"is_managed_bucket_notifications_enabled"`
	BucketName                          types.String          `tfsdk:"bucket_name"`
	PrefixLogTypes                      []PrefixLogTypesModel `tfsdk:"prefix_log_types"`
	Id                                  types.String          `tfsdk:"id"`
}

type PrefixLogTypesModel struct {
	ExcludedPrefixes []types.String `tfsdk:"excluded_prefixes"`
	LogTypes         []types.String `tfsdk:"log_types"`
	Prefix           types.String   `tfsdk:"prefix"`
}

func (r *S3SourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_source"
}

func (r *S3SourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Represents an S3 Log Source in Panther",

		Attributes: map[string]schema.Attribute{
			"aws_account_id": schema.StringAttribute{
				MarkdownDescription: "Example configurable attribute",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"kms_key_arn": schema.StringAttribute{
				MarkdownDescription: "Example configurable attribute",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Example configurable attribute",
				Required:            true,
			},
			"log_processing_role_arn": schema.StringAttribute{
				MarkdownDescription: "Example configurable attribute",
				Required:            true,
			},
			// TODO oneof validation
			// https://github.com/hashicorp/terraform-plugin-framework-validators/blob/main/stringvalidator/one_of.go
			"log_stream_type": schema.StringAttribute{
				MarkdownDescription: "Example configurable attribute",
				Required:            true,
			},
			"is_managed_bucket_notifications_enabled": schema.BoolAttribute{
				MarkdownDescription: "Example configurable attribute",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"bucket_name": schema.StringAttribute{
				MarkdownDescription: "Example configurable attribute",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"prefix_log_types": schema.ListNestedAttribute{
				MarkdownDescription: "Example configurable attribute",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"excluded_prefixes": schema.ListAttribute{
							ElementType:         types.StringType,
							Required:            true,
							MarkdownDescription: "",
						},
						"log_types": schema.ListAttribute{
							ElementType:         types.StringType,
							Required:            true,
							MarkdownDescription: "",
						},
						"prefix": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "",
						},
					},
				},
				Optional: true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Required:            false,
				Optional:            false,
				MarkdownDescription: "Example identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *S3SourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *S3SourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *S3SourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make the GraphQL mutation to create the resource
	output, err := r.client.CreateS3Source(ctx, client.CreateS3SourceInput{
		AwsAccountID:               data.AWSAccountID.ValueString(),
		KmsKey:                     data.KMSKeyARN.ValueString(),
		Label:                      data.Name.ValueString(),
		LogProcessingRole:          data.LogProcessingRoleARN.ValueString(),
		LogStreamType:              client.LogStreamType(data.LogStreamType.ValueString()),
		ManagedBucketNotifications: data.IsManagedBucketNotificationsEnabled.ValueBool(),
		S3Bucket:                   data.BucketName.ValueString(),
		S3PrefixLogTypes:           PrefixLogTypesToInput(data.PrefixLogTypes),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating S3 Source",
			"Could not create S3 Source, unexpected error: "+err.Error(),
		)
		return
	}
	data.Id = types.StringValue(output.LogSource.IntegrationID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3SourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *S3SourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3SourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *S3SourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3SourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *S3SourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *S3SourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convert terraform model to Panther client input
func PrefixLogTypesToInput(prefixLogTypes []PrefixLogTypesModel) []client.S3PrefixLogTypesInput {
	result := []client.S3PrefixLogTypesInput{}
	for _, p := range prefixLogTypes {
		var excluded, logTypes []string
		for _, v := range p.ExcludedPrefixes {
			excluded = append(excluded, v.ValueString())
		}
		for _, v := range p.LogTypes {
			logTypes = append(excluded, v.ValueString())
		}
		result = append(result,
			client.S3PrefixLogTypesInput{
				ExcludedPrefixes: excluded,
				Prefix:           p.Prefix.ValueString(),
				LogTypes:         logTypes,
			})
	}
	return result
}
