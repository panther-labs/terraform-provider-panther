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

	"terraform-provider-panther/internal/client/panther"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// providerClients extracts *panther.ProviderClients from the Terraform provider data.
// Returns nil if provider data is not yet available (during early lifecycle).
func providerClients(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *panther.ProviderClients {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*panther.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *panther.ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return c
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
