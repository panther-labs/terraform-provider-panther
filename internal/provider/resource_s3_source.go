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
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	AWSAccountID                             types.String          `tfsdk:"aws_account_id"`
	KMSKeyARN                                types.String          `tfsdk:"kms_key_arn"`
	Name                                     types.String          `tfsdk:"name"`
	LogProcessingRoleARN                     types.String          `tfsdk:"log_processing_role_arn"`
	LogStreamType                            types.String          `tfsdk:"log_stream_type"`
	PantherManagedBucketNotificationsEnabled types.Bool            `tfsdk:"panther_managed_bucket_notifications_enabled"`
	BucketName                               types.String          `tfsdk:"bucket_name"`
	PrefixLogTypes                           []PrefixLogTypesModel `tfsdk:"prefix_log_types"`
	Id                                       types.String          `tfsdk:"id"`
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
				Description:   "The ID of the AWS Account where the S3 Bucket is located.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"kms_key_arn": schema.StringAttribute{
				Description: "The KMS key ARN used to access the S3 Bucket.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"name": schema.StringAttribute{
				Description: "The display name of the S3 Log Source integration.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[0-9a-zA-Z- ]+$"),
						"must only include alphanumeric characters, dashes and spaces",
					),
					stringvalidator.LengthAtMost(32),
				},
			},
			"log_processing_role_arn": schema.StringAttribute{
				Description: "The AWS Role used to access the S3 Bucket.",
				Required:    true,
			},
			"log_stream_type": schema.StringAttribute{
				Description: "The format of the log files being ingested.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("Auto", "Lines", "JSON", "JsonArray", "CloudWatchLogs"),
				},
			},
			"panther_managed_bucket_notifications_enabled": schema.BoolAttribute{
				MarkdownDescription: `True if bucket notifications are being managed by Panther.  __This will cause Panther to create additional infrastructure in your AWS account.__ \
To manage the notification-related infrastructure through terraform, refer to [this example](https://github.com/panther-labs/panther-auxiliary/tree/main/terraform/panther_log_processing_notifications).`,
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"bucket_name": schema.StringAttribute{
				Description:   "The name of the S3 Bucket where logs will be ingested from.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"prefix_log_types": schema.ListNestedAttribute{
				Description: "The configured mapping of prefixes to log types.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"excluded_prefixes": schema.ListAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: "S3 Prefixes to be excluded from log type mapping.",
						},
						"log_types": schema.ListAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: "List of log types that map to the S3 Prefix.",
						},
						"prefix": schema.StringAttribute{
							Required:    true,
							Description: "S3 Prefix to map Log Types to.",
						},
					},
				},
				Required: true,
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

	c, ok := req.ProviderData.(client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = c
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
		LogStreamType:              data.LogStreamType.ValueString(),
		ManagedBucketNotifications: data.PantherManagedBucketNotificationsEnabled.ValueBool(),
		S3Bucket:                   data.BucketName.ValueString(),
		S3PrefixLogTypes:           prefixLogTypesToInput(data.PrefixLogTypes),
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

	source, err := r.client.GetS3Source(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading S3 Source",
			"Could not read S3 Source, unexpected error: "+err.Error(),
		)
		return
	}

	data.Id = types.StringValue(source.IntegrationID)
	data.AWSAccountID = types.StringValue(source.AwsAccountID)
	data.KMSKeyARN = types.StringValue(source.KmsKey)
	data.Name = types.StringValue(source.IntegrationLabel)
	data.LogProcessingRoleARN = types.StringPointerValue(source.LogProcessingRole)
	data.LogStreamType = types.StringPointerValue(source.LogStreamType)
	data.PantherManagedBucketNotificationsEnabled = types.BoolValue(source.ManagedBucketNotifications)
	data.BucketName = types.StringValue(source.S3Bucket)
	data.PrefixLogTypes = prefixLogTypesToModel(source.S3PrefixLogTypes)

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

	_, err := r.client.UpdateS3Source(ctx, client.UpdateS3SourceInput{
		ID:                         data.Id.ValueString(),
		KmsKey:                     data.KMSKeyARN.ValueString(),
		Label:                      data.Name.ValueString(),
		LogProcessingRole:          data.LogProcessingRoleARN.ValueString(),
		LogStreamType:              data.LogStreamType.ValueString(),
		ManagedBucketNotifications: data.PantherManagedBucketNotificationsEnabled.ValueBool(),
		S3PrefixLogTypes:           prefixLogTypesToInput(data.PrefixLogTypes),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating S3 Source",
			"Could not update S3 Source, unexpected error: "+err.Error(),
		)
		return
	}

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

	_, err := r.client.DeleteSource(ctx, client.DeleteSourceInput{ID: data.Id.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting S3 Source",
			"Could not delete S3 Source, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *S3SourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convert terraform model to Panther client input
func prefixLogTypesToInput(prefixLogTypes []PrefixLogTypesModel) []client.S3PrefixLogTypesInput {
	result := []client.S3PrefixLogTypesInput{}
	for _, p := range prefixLogTypes {
		excluded := []string{}
		logTypes := []string{}
		for _, v := range p.ExcludedPrefixes {
			excluded = append(excluded, v.ValueString())
		}
		for _, v := range p.LogTypes {
			logTypes = append(logTypes, v.ValueString())
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

// convert Panther client output to terraform model
func prefixLogTypesToModel(prefixLogTypes []client.S3PrefixLogTypes) []PrefixLogTypesModel {
	result := []PrefixLogTypesModel{}
	for _, p := range prefixLogTypes {
		excluded := []types.String{}
		logTypes := []types.String{}
		for _, v := range p.ExcludedPrefixes {
			excluded = append(excluded, types.StringValue(v))
		}
		for _, v := range p.LogTypes {
			logTypes = append(logTypes, types.StringValue(v))
		}
		result = append(result,
			PrefixLogTypesModel{
				ExcludedPrefixes: excluded,
				Prefix:           types.StringValue(p.Prefix),
				LogTypes:         logTypes,
			})
	}
	return result
}
