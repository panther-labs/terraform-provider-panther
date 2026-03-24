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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

// handleReadError handles API errors in Read operations.
// Returns true if the error was handled (caller should return).
// 404 errors remove the resource from state (drift detection); other errors add a diagnostic.
func handleReadError(ctx context.Context, resp *resource.ReadResponse, resourceName, id string, err error) bool {
	if err == nil {
		return false
	}
	if client.IsNotFound(err) {
		tflog.Warn(ctx, fmt.Sprintf("%s %s not found, removing from state", resourceName, id))
		resp.State.RemoveResource(ctx)
		return true
	}
	resp.Diagnostics.AddError(
		fmt.Sprintf("Error reading %s", resourceName),
		fmt.Sprintf("Could not read %s (id=%s): %s", resourceName, id, err.Error()),
	)
	return true
}

// handleCreateError handles API errors in Create operations.
// Returns true if the error was handled (caller should return).
// 409 conflicts produce a user-friendly message guiding toward `terraform import`.
func handleCreateError(resp *resource.CreateResponse, resourceName string, err error) bool {
	if err == nil {
		return false
	}
	if client.IsConflict(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s already exists", resourceName),
			fmt.Sprintf("A %s with these attributes already exists. "+
				"Use `terraform import` to adopt the existing resource into Terraform state.\n\nAPI error: %s",
				resourceName, err.Error()),
		)
		return true
	}
	resp.Diagnostics.AddError(
		fmt.Sprintf("Error creating %s", resourceName),
		fmt.Sprintf("Could not create %s: %s", resourceName, err.Error()),
	)
	return true
}

// handleUpdateError handles API errors in Update operations.
// Returns true if the error was handled (caller should return).
// 409 conflicts produce a user-friendly message about duplicate labels.
func handleUpdateError(resp *resource.UpdateResponse, resourceName, id string, err error) bool {
	if err == nil {
		return false
	}
	if client.IsConflict(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Conflict updating %s", resourceName),
			fmt.Sprintf("Cannot update %s (id=%s): the update conflicts with an existing resource.\n\nAPI error: %s",
				resourceName, id, err.Error()),
		)
		return true
	}
	resp.Diagnostics.AddError(
		fmt.Sprintf("Error updating %s", resourceName),
		fmt.Sprintf("Could not update %s (id=%s): %s", resourceName, id, err.Error()),
	)
	return true
}

// handleDeleteError handles API errors in Delete operations.
// Returns true if the error was handled (caller should return).
// 404 is treated as success (resource already deleted).
func handleDeleteError(resp *resource.DeleteResponse, resourceName, id string, err error) bool {
	if err == nil || client.IsNotFound(err) {
		return false
	}
	resp.Diagnostics.AddError(
		fmt.Sprintf("Error deleting %s", resourceName),
		fmt.Sprintf("Could not delete %s (id=%s): %s", resourceName, id, err.Error()),
	)
	return true
}

// patchIDAttribute adds UseStateForUnknown to the generated "id" attribute.
// Every resource needs this because the code generator doesn't support plan modifiers.
func patchIDAttribute(s *schema.Schema) {
	raw, ok := s.Attributes["id"]
	if !ok {
		return
	}
	idAttr, ok := raw.(schema.StringAttribute)
	if !ok {
		return
	}
	idAttr.PlanModifiers = append(idAttr.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	s.Attributes["id"] = idAttr
}

func convertLogTypes(ctx context.Context, logTypes types.List, diagnostics diag.Diagnostics) []string {
	var result []string
	diagnostics.Append(logTypes.ElementsAs(ctx, &result, false)...)
	return result
}

func convertFromLogTypes(ctx context.Context, logTypes []string, diagnostics diag.Diagnostics) types.List {
	from, d := types.ListValueFrom(ctx, types.StringType, logTypes)
	diagnostics.Append(d...)
	return from
}
