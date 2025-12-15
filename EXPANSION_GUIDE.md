# Panther Terraform Provider Expansion Guide

This document outlines the current state of the Panther Labs Terraform provider and what remains to be implemented.

## Current Implementation Status

### ✅ Fully Implemented Resources (6/9)

**GraphQL-Based Resources:**

- **`panther_s3_source`** - S3 log source integration ✅
- **`panther_cloud_account`** - AWS cloud account integration with exclusion features ✅  
- **`panther_user`** - User management and invitation ✅
- **`panther_role`** - Role-based access control ✅

**REST-Based Resources:**

- **`panther_httpsource`** - HTTP log source integration ✅
- **`panther_rule`** - Detection rule management with generated schema ✅

### 🚧 Partially Implemented Resources (3/9)

The following resources have been scaffolded with generated code but need full CRUD implementation:

**REST-Based Resources (Generated but not implemented):**

- **`panther_policy`** - Compliance policy management 🚧
- **`panther_datamodel`** - Custom log parsing schemas 🚧  
- **`panther_global`** - Global variables and helper functions 🚧

## What's Left To Do

### Phase 1: Complete the Remaining REST Resources

The groundwork is already in place. For each remaining resource, you need to:

#### 1. Policy Resource (`panther_policy`)

**Generated Files Available:**

- `internal/provider/resource_policy/` - Generated schema and models
- REST client methods already implemented in `internal/client/panther/panther.go`

**Implementation Required:**

```bash
# 1. Create the main resource file
cp internal/provider/resource_rule.go internal/provider/resource_policy.go

# 2. Update the resource implementation to use policy-specific types
# - Replace "rule" with "policy" throughout
# - Update schema to use generated PolicyResourceSchema  
# - Update CRUD operations to use policy client methods

# 3. Create test file
cp internal/provider/resource_rule_test.go internal/provider/resource_policy_test.go

# 4. Register in provider.go (uncomment the line)
```

**Client Methods Available:**

- `CreatePolicy(ctx, CreatePolicyInput) (Policy, error)`
- `UpdatePolicy(ctx, UpdatePolicyInput) (Policy, error)`  
- `GetPolicy(ctx, id string) (Policy, error)`
- `DeletePolicy(ctx, id string) error`

#### 2. Data Model Resource (`panther_datamodel`)

**Generated Files Available:**

- `internal/provider/resource_datamodel/` - Generated schema and models
- REST client methods already implemented

**Implementation Required:**

- Follow same pattern as policy resource above
- Use DataModel types and client methods
- Data models are used for custom log parsing schemas

**Client Methods Available:**

- `CreateDataModel(ctx, CreateDataModelInput) (DataModel, error)`
- `UpdateDataModel(ctx, UpdateDataModelInput) (DataModel, error)`
- `GetDataModel(ctx, id string) (DataModel, error)`
- `DeleteDataModel(ctx, id string) error`

#### 3. Global Resource (`panther_global`)

**Generated Files Available:**

- `internal/provider/resource_global/` - Generated schema and models  
- REST client methods already implemented

**Implementation Required:**

- Follow same pattern as policy resource above
- Use Global types and client methods
- Globals are helper functions used across rules and policies

**Client Methods Available:**

- `CreateGlobal(ctx, CreateGlobalInput) (Global, error)`
- `UpdateGlobal(ctx, UpdateGlobalInput) (Global, error)`
- `GetGlobal(ctx, id string) (Global, error)`
- `DeleteGlobal(ctx, id string) error`

### Phase 2: Register Resources in Provider

Once implemented, uncomment these lines in `internal/provider/provider.go`:

```go
// Uncomment these as you implement each resource:
// "panther_policy":    func() resource.Resource { return NewPolicyResource() },
// "panther_datamodel": func() resource.Resource { return NewDataModelResource() },  
// "panther_global":    func() resource.Resource { return NewGlobalResource() },
```

### Phase 3: Testing and Documentation

For each implemented resource:

1. **Run Tests:**

   ```bash
   go test ./internal/provider -run TestPolicyResource -v
   go test ./internal/provider -run TestDataModelResource -v  
   go test ./internal/provider -run TestGlobalResource -v
   ```

2. **Generate Documentation:**

   ```bash
   go generate
   ```

3. **Run Full Test Suite:**

   ```bash
   make testacc
   ```

## Implementation Template

Here's the exact pattern to follow for each remaining resource:

### 1. Main Resource File

```go
// internal/provider/resource_policy.go
package provider

import (
    "context" 
    "fmt"
    "terraform-provider-panther/internal/client"
    "terraform-provider-panther/internal/client/panther"
    "terraform-provider-panther/internal/provider/resource_policy" // Generated

    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

func NewPolicyResource() resource.Resource {
    return &policyResource{}
}

type policyResource struct {
    client client.RestClient
}

func (r *policyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    // Use the generated schema
    generatedSchema := resource_policy.PolicyResourceSchema(ctx)
    
    // Add the ID field with UseStateForUnknown as required
    generatedSchema.Attributes["id"] = schema.StringAttribute{
        Computed: true,
        PlanModifiers: []planmodifier.String{
            stringplanmodifier.UseStateForUnknown(),
        },
    }
    resp.Schema = generatedSchema
}

func (r *policyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    c, ok := req.ProviderData.(*panther.APIClient)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *panther.APIClient, got: %T.", req.ProviderData),
        )
        return
    }

    r.client = c.RestClient
}

// Implement Create, Read, Update, Delete, ImportState following resource_rule.go pattern
```

### 2. Test File

```go
// internal/provider/resource_policy_test.go  
package provider

import (
    "fmt"
    "strings" 
    "testing"

    "github.com/google/uuid"
    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPolicyResource(t *testing.T) {
    policyName := strings.ReplaceAll(uuid.NewString(), "-", "")
    // Follow the pattern from resource_rule_test.go
}
```

## Current Architecture Status

### ✅ Complete Architecture Components

- **Dual Client System**: GraphQL + REST clients working perfectly
- **Generated Schema System**: OpenAPI integration working for REST resources  
- **State Management**: Proper null vs empty array handling
- **URL Flexibility**: Supports both AWS API Gateway and direct Panther URLs
- **Test Framework**: All 8/8 current tests passing
- **Documentation Generation**: Automated doc generation working
- **Import Support**: All resources support Terraform import

### 🔧 Ready Infrastructure

- **Client Methods**: All REST client methods implemented and tested
- **Type Definitions**: Complete type system for all resources
- **Generated Code**: Schemas and models generated for remaining resources
- **Test Templates**: Established patterns to follow
- **Provider Registration**: Framework ready for new resources

## Estimated Implementation Time

Each remaining resource should take approximately **2-4 hours** to implement following the established patterns:

- **Policy Resource**: ~3 hours (compliance policies are more complex)
- **DataModel Resource**: ~2 hours (straightforward schema management)  
- **Global Resource**: ~2 hours (simple helper functions)

**Total remaining work**: ~7 hours to complete all Terraform resources

## Implementation Priority

**Highest Priority:**

1. **Policy Resource** - Compliance policies are core security functionality

**Medium Priority:**  
2. **DataModel Resource** - Custom log parsing is important for advanced users

**Lower Priority:**
3. **Global Resource** - Helper functions, nice-to-have but not critical

The foundation is solid and all the infrastructure is in place. Following the established patterns, implementing the remaining resources should be straightforward!
