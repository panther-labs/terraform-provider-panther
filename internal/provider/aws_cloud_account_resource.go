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
	"terraform-provider-panther/internal/provider/resource_aws_cloud_account"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const awsCloudAccountPath = "/cloud-accounts/aws"

var (
	auditRoleARNRegex = regexp.MustCompile(`^arn:aws[a-z-]*:iam::\d{12}:role/.+$`)
	awsRegionRegex    = regexp.MustCompile(`^[a-z]{2}(?:-gov)?-[a-z]+-\d+$`)
)

var (
	_ resource.Resource                = (*awsCloudAccountResource)(nil)
	_ resource.ResourceWithConfigure   = (*awsCloudAccountResource)(nil)
	_ resource.ResourceWithImportState = (*awsCloudAccountResource)(nil)
)

func NewAwsCloudAccountResource() resource.Resource {
	return &awsCloudAccountResource{}
}

type awsCloudAccountResource struct {
	rest *client.RESTClient
}

func (r *awsCloudAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_cloud_account"
}

func (r *awsCloudAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_aws_cloud_account.AwsCloudAccountResourceSchema(ctx)
	resp.Schema.MarkdownDescription = "Manages an AWS Cloud Account integration for Panther's compliance scanner.\n\n" +
		"**Caveat — scan interval:** the REST API hardcodes a 24-hour scan interval " +
		"(1440 minutes) on every create/update and does not expose `scanIntervalMins`. " +
		"Custom intervals set out-of-band (Panther UI or GraphQL) will be silently " +
		"reset on the first `terraform apply` after import."

	applySchemaOverrides(&resp.Schema, []SchemaOverride{
		{Name: "id", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		// aws_account_id is immutable server-side (ModifyCloudAccount drops it).
		{Name: "aws_account_id", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
	})

	// audit_role must be Required; the OpenAPI spec omits it from the
	// AWSScanConfig `required:` array because of a Goa v3 codegen bug on
	// nested types.
	if scanCfg, ok := resp.Schema.Attributes["aws_scan_config"].(schema.SingleNestedAttribute); ok {
		if auditRole, ok := scanCfg.Attributes["audit_role"].(schema.StringAttribute); ok {
			auditRole.Required = true
			auditRole.Optional = false
			auditRole.Computed = false
			scanCfg.Attributes["audit_role"] = auditRole
		}
		resp.Schema.Attributes["aws_scan_config"] = scanCfg
	}

	setEmptyListDefault(&resp.Schema, "region_ignore_list")
	setEmptyListDefault(&resp.Schema, "resource_type_ignore_list")
	setEmptyListDefault(&resp.Schema, "resource_regex_ignore_list")

	// Validators the OpenAPI codegen can't emit: audit_role ARN (nested-type
	// codegen bug), region_ignore_list items pattern (current codegen doesn't
	// lift items.pattern), and resource_regex_ignore_list "compiles as regex"
	// (no OpenAPI primitive).
	addNestedStringValidator(&resp.Schema, "aws_scan_config", "audit_role",
		stringvalidator.RegexMatches(auditRoleARNRegex,
			"must be a valid IAM role ARN (e.g. arn:aws:iam::123456789012:role/PantherAuditRole)"),
	)
	addListElementValidator(&resp.Schema, "region_ignore_list",
		stringvalidator.RegexMatches(awsRegionRegex,
			"must be a valid AWS region code (e.g. us-east-1, us-gov-west-1)"),
	)
	addListElementValidator(&resp.Schema, "resource_regex_ignore_list", compilesAsRegex{})
}

func (r *awsCloudAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.rest = restClient(req, resp)
}

func (r *awsCloudAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_aws_cloud_account.AwsCloudAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.AwsCloudAccountInput{
		IntegrationLabel:        data.IntegrationLabel.ValueString(),
		AwsAccountId:            data.AwsAccountId.ValueString(),
		AwsScanConfig:           client.AwsScanConfig{AuditRole: data.AwsScanConfig.AuditRole.ValueString()},
		RegionIgnoreList:        convertLogTypes(ctx, data.RegionIgnoreList, &resp.Diagnostics),
		ResourceTypeIgnoreList:  convertLogTypes(ctx, data.ResourceTypeIgnoreList, &resp.Diagnostics),
		ResourceRegexIgnoreList: convertLogTypes(ctx, data.ResourceRegexIgnoreList, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := client.RestDo[client.AwsCloudAccount](ctx, r.rest, http.MethodPost, awsCloudAccountPath, input)
	if handleCreateError(resp, "AWS Cloud Account", err) {
		return
	}
	tflog.Debug(ctx, "Created AWS Cloud Account", map[string]any{"id": out.IntegrationId})

	data.Id = types.StringValue(out.IntegrationId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *awsCloudAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_aws_cloud_account.AwsCloudAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := client.RestDo[client.AwsCloudAccount](ctx, r.rest, http.MethodGet, awsCloudAccountPath+"/"+data.Id.ValueString(), nil)
	if handleReadError(ctx, resp, "AWS Cloud Account", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Read AWS Cloud Account", map[string]any{"id": out.IntegrationId})

	data.Id = types.StringValue(out.IntegrationId)
	data.IntegrationLabel = types.StringValue(out.IntegrationLabel)
	data.AwsAccountId = types.StringValue(out.AwsAccountId)
	scanCfg, scanCfgDiags := resource_aws_cloud_account.NewAwsScanConfigValue(
		resource_aws_cloud_account.AwsScanConfigValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"audit_role": types.StringValue(out.AwsScanConfig.AuditRole),
		},
	)
	resp.Diagnostics.Append(scanCfgDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AwsScanConfig = scanCfg
	data.RegionIgnoreList = convertFromLogTypes(ctx, out.RegionIgnoreList, &resp.Diagnostics)
	data.ResourceTypeIgnoreList = convertFromLogTypes(ctx, out.ResourceTypeIgnoreList, &resp.Diagnostics)
	data.ResourceRegexIgnoreList = convertFromLogTypes(ctx, out.ResourceRegexIgnoreList, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *awsCloudAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_aws_cloud_account.AwsCloudAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// AwsAccountId omitted: ModifyCloudAccount drops it; the zero value is
	// elided by `omitempty`. RequiresReplace handles config diffs.
	input := client.AwsCloudAccountInput{
		IntegrationLabel:        data.IntegrationLabel.ValueString(),
		AwsScanConfig:           client.AwsScanConfig{AuditRole: data.AwsScanConfig.AuditRole.ValueString()},
		RegionIgnoreList:        convertLogTypes(ctx, data.RegionIgnoreList, &resp.Diagnostics),
		ResourceTypeIgnoreList:  convertLogTypes(ctx, data.ResourceTypeIgnoreList, &resp.Diagnostics),
		ResourceRegexIgnoreList: convertLogTypes(ctx, data.ResourceRegexIgnoreList, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := client.RestDo[client.AwsCloudAccount](ctx, r.rest, http.MethodPut, awsCloudAccountPath+"/"+data.Id.ValueString(), input)
	if handleUpdateError(ctx, resp, "AWS Cloud Account", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Updated AWS Cloud Account", map[string]any{"id": data.Id.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *awsCloudAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_aws_cloud_account.AwsCloudAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.RestDelete(ctx, r.rest, awsCloudAccountPath+"/"+data.Id.ValueString())
	if handleDeleteError(resp, "AWS Cloud Account", data.Id.ValueString(), err) {
		return
	}
	tflog.Debug(ctx, "Deleted AWS Cloud Account", map[string]any{"id": data.Id.ValueString()})
}

func (r *awsCloudAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
