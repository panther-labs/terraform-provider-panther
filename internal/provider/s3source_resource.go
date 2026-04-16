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
	"regexp"
	"terraform-provider-panther/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const s3SourcePath = "/log-sources/s3"

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = (*S3SourceResource)(nil)
	_ resource.ResourceWithImportState = (*S3SourceResource)(nil)
	_ resource.ResourceWithConfigure   = (*S3SourceResource)(nil)
)

func NewS3SourceResource() resource.Resource {
	return &S3SourceResource{}
}

type S3SourceResource struct {
	rest *client.RESTClient
}

// S3SourceResourceModel describes the resource data model.
type S3SourceResourceModel struct {
	AWSAccountID                             types.String          `tfsdk:"aws_account_id"`
	KMSKeyARN                                types.String          `tfsdk:"kms_key_arn"`
	Name                                     types.String          `tfsdk:"name"`
	LogProcessingRoleARN                     types.String          `tfsdk:"log_processing_role_arn"`
	LogStreamType                            types.String          `tfsdk:"log_stream_type"`
	LogStreamTypeOptions                     types.Object          `tfsdk:"log_stream_type_options"`
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

func (r *S3SourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
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
				Description: "The format of the log files being ingested. Supported log stream types: Auto, JSON, JsonArray, Lines, CloudWatchLogs, XML",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("Auto", "Lines", "JSON", "JsonArray", "CloudWatchLogs", "XML"),
				},
			},
			"log_stream_type_options": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"json_array_envelope_field": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Description: "Path to the JSON array field to extract records from. Only applicable when log_stream_type is JsonArray.",
					},
					"retain_envelope_fields": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Preserve CloudWatch Logs envelope metadata (accountId, logGroup, subscriptionFilters) in a p_header column. Only applicable when log_stream_type is CloudWatchLogs.",
					},
					"xml_root_element": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Description: "Root element wrapping XML events. Only applicable when log_stream_type is XML.",
					},
				},
				Optional: true,
				Computed: true,
				Default: objectdefault.StaticValue(types.ObjectNull(map[string]attr.Type{
					"json_array_envelope_field": types.StringType,
					"retain_envelope_fields":    types.BoolType,
					"xml_root_element":          types.StringType,
				})),
			},
			"panther_managed_bucket_notifications_enabled": schema.BoolAttribute{
				MarkdownDescription: `True if bucket notifications are being managed by Panther.  __This will cause Panther to create additional infrastructure in your AWS account.__ \
To manage the notification-related infrastructure through terraform, refer to [this example](https://github.com/panther-labs/panther-auxiliary/tree/main/terraform/panther_log_processing_notifications).`,
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			// RequiresReplace: s3Bucket is immutable in the API (excluded from PUT schema).
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
				MarkdownDescription: "The unique identifier of the S3 log source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *S3SourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *S3SourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *S3SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.S3SourceCreateInput{
		AwsAccountId:               data.AWSAccountID.ValueString(),
		IntegrationLabel:           data.Name.ValueString(),
		S3Bucket:                   data.BucketName.ValueString(),
		KmsKey:                     data.KMSKeyARN.ValueString(),
		LogProcessingRole:          data.LogProcessingRoleARN.ValueString(),
		LogStreamType:              data.LogStreamType.ValueString(),
		LogStreamTypeOptions:       s3LogStreamTypeOptions(data.LogStreamTypeOptions),
		ManagedBucketNotifications: data.PantherManagedBucketNotificationsEnabled.ValueBool(),
		S3PrefixLogTypes:           prefixLogTypesToInput(data.PrefixLogTypes),
	}

	s3Source, err := client.RestDo[client.S3Source](ctx, r.rest, http.MethodPost, s3SourcePath, input)
	if handleCreateError(resp, "S3 Source", err) {
		return
	}
	tflog.Debug(ctx, "Created S3 Source", map[string]any{"id": s3Source.IntegrationId})

	data.Id = types.StringValue(s3Source.IntegrationId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3SourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *S3SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s3Source, err := client.RestDo[client.S3Source](ctx, r.rest, http.MethodGet, s3SourcePath+"/"+data.Id.ValueString(), nil)
	if handleReadError(ctx, resp, "S3 Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Read S3 Source", map[string]any{"id": s3Source.IntegrationId})

	data.Id = types.StringValue(s3Source.IntegrationId)
	data.AWSAccountID = types.StringValue(s3Source.AwsAccountId)
	data.KMSKeyARN = types.StringValue(s3Source.KmsKey)
	data.Name = types.StringValue(s3Source.IntegrationLabel)
	data.LogProcessingRoleARN = types.StringValue(s3Source.LogProcessingRole)
	data.LogStreamType = types.StringValue(s3Source.LogStreamType)
	data.PantherManagedBucketNotificationsEnabled = types.BoolValue(s3Source.ManagedBucketNotifications)
	data.BucketName = types.StringValue(s3Source.S3Bucket)
	data.PrefixLogTypes = prefixLogTypesToModel(s3Source.S3PrefixLogTypes)

	logStreamTypeOptionAttrTypes := map[string]attr.Type{
		"json_array_envelope_field": types.StringType,
		"retain_envelope_fields":    types.BoolType,
		"xml_root_element":          types.StringType,
	}
	if s3Source.LogStreamTypeOptions != nil && (s3Source.LogStreamTypeOptions.JsonArrayEnvelopeField != "" ||
		s3Source.LogStreamTypeOptions.XmlRootElement != "" || s3Source.LogStreamTypeOptions.RetainEnvelopeFields) {
		attributeValues := map[string]attr.Value{
			"json_array_envelope_field": types.StringValue(s3Source.LogStreamTypeOptions.JsonArrayEnvelopeField),
			"retain_envelope_fields":    types.BoolValue(s3Source.LogStreamTypeOptions.RetainEnvelopeFields),
			"xml_root_element":          types.StringValue(s3Source.LogStreamTypeOptions.XmlRootElement),
		}
		objectValue, diags := basetypes.NewObjectValue(logStreamTypeOptionAttrTypes, attributeValues)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.LogStreamTypeOptions = objectValue
	} else {
		data.LogStreamTypeOptions = types.ObjectNull(logStreamTypeOptionAttrTypes)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3SourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *S3SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.S3SourceUpdateInput{
		IntegrationLabel:           data.Name.ValueString(),
		KmsKey:                     data.KMSKeyARN.ValueString(),
		LogProcessingRole:          data.LogProcessingRoleARN.ValueString(),
		LogStreamType:              data.LogStreamType.ValueString(),
		LogStreamTypeOptions:       s3LogStreamTypeOptions(data.LogStreamTypeOptions),
		ManagedBucketNotifications: data.PantherManagedBucketNotificationsEnabled.ValueBool(),
		S3PrefixLogTypes:           prefixLogTypesToInput(data.PrefixLogTypes),
	}

	_, err := client.RestDo[client.S3Source](ctx, r.rest, http.MethodPut, s3SourcePath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(resp, "S3 Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated S3 Source", map[string]any{"id": data.Id.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3SourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *S3SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, s3SourcePath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "S3 Source", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Deleted S3 Source", map[string]any{"id": data.Id.ValueString()})
}

func (r *S3SourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// s3LogStreamTypeOptions extracts LogStreamTypeOptions from a Terraform nested object.
// Returns nil when the object is null/unknown or when all inner fields are null/unknown.
func s3LogStreamTypeOptions(opts types.Object) *client.LogStreamTypeOptions {
	if opts.IsNull() || opts.IsUnknown() {
		return nil
	}
	result := &client.LogStreamTypeOptions{}
	attrs := opts.Attributes()
	if val, ok := attrs["json_array_envelope_field"]; ok && !val.IsNull() && !val.IsUnknown() {
		if sv, ok := val.(types.String); ok {
			result.JsonArrayEnvelopeField = sv.ValueString()
		}
	}
	if val, ok := attrs["retain_envelope_fields"]; ok && !val.IsNull() && !val.IsUnknown() {
		if bv, ok := val.(types.Bool); ok {
			result.RetainEnvelopeFields = bv.ValueBool()
		}
	}
	if val, ok := attrs["xml_root_element"]; ok && !val.IsNull() && !val.IsUnknown() {
		if sv, ok := val.(types.String); ok {
			result.XmlRootElement = sv.ValueString()
		}
	}
	// Return nil if all fields are zero — avoids sending empty {} to the API.
	if result.JsonArrayEnvelopeField == "" && !result.RetainEnvelopeFields && result.XmlRootElement == "" {
		return nil
	}
	return result
}

// prefixLogTypesToInput converts the Terraform model to REST API input structs.
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

// prefixLogTypesToModel converts REST API response prefix mappings to the Terraform model.
func prefixLogTypesToModel(prefixLogTypes []client.S3PrefixLogTypesInput) []PrefixLogTypesModel {
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
