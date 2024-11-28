// Code generated by terraform-plugin-framework-generator DO NOT EDIT.

package resource_httpsource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func HttpsourceResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "ID of the http source to fetch",
				MarkdownDescription: "ID of the http source to fetch",
			},
			"integration_label": schema.StringAttribute{
				Required:            true,
				Description:         "The id of the data model",
				MarkdownDescription: "The id of the data model",
			},
			"log_stream_type": schema.StringAttribute{
				Required:            true,
				Description:         "The log stream types",
				MarkdownDescription: "The log stream types",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"Auto",
						"CloudWatchLogs",
						"JSON",
						"JsonArray",
						"Lines",
					),
				},
			},
			"log_types": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				Description:         "The log types of the integration",
				MarkdownDescription: "The log types of the integration",
			},
			"security_alg": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The authentication algorithm of the http source",
				MarkdownDescription: "The authentication algorithm of the http source",
			},
			"security_header_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The security header key of the http source",
				MarkdownDescription: "The security header key of the http source",
			},
			"security_password": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The authentication header password of the http source",
				MarkdownDescription: "The authentication header password of the http source",
			},
			"security_secret_value": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The header secret value of the http source",
				MarkdownDescription: "The header secret value of the http source",
			},
			"security_type": schema.StringAttribute{
				Required:            true,
				Description:         "The security type of the http endpoint",
				MarkdownDescription: "The security type of the http endpoint",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"SharedSecret",
						"HMAC",
						"Bearer",
						"Basic",
						"None",
					),
				},
			},
			"security_username": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The authentication header username of the http source",
				MarkdownDescription: "The authentication header username of the http source",
			},
		},
	}
}

type HttpsourceModel struct {
	Id                  types.String `tfsdk:"id"`
	IntegrationLabel    types.String `tfsdk:"integration_label"`
	LogStreamType       types.String `tfsdk:"log_stream_type"`
	LogTypes            types.List   `tfsdk:"log_types"`
	SecurityAlg         types.String `tfsdk:"security_alg"`
	SecurityHeaderKey   types.String `tfsdk:"security_header_key"`
	SecurityPassword    types.String `tfsdk:"security_password"`
	SecuritySecretValue types.String `tfsdk:"security_secret_value"`
	SecurityType        types.String `tfsdk:"security_type"`
	SecurityUsername    types.String `tfsdk:"security_username"`
}
