# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Installation

- `go install` - Build and install the provider binary to `$GOPATH/bin`
- `go generate` - Generate or update documentation

### Testing

- `make testacc` - Run full acceptance test suite (requires `PANTHER_API_URL` and `PANTHER_API_TOKEN` environment variables)
- `go test ./internal/... -v -timeout 120m` - Run unit tests with extended timeout
- `go test ./internal/provider -run TestCloudAccount -v` - Run specific resource tests

### Code Generation

The provider uses Terraform's framework code generator and OpenAPI generator for resource scaffolding:

1. **Download OpenAPI spec**: `curl -o panther_openapi_spec.yaml https://api.panther.dev/public/openapi.yaml`
2. **Update provider-code-spec.json**: `tfplugingen-openapi generate --config ./generator_config.yml --output ./provider-code-spec.json panther_openapi_spec.yaml`
3. **Generate resource models/schema**: `tfplugingen-framework generate resources --input ./provider-code-spec.json --output ./internal/provider`

### Dependencies

- `go get {package}` followed by `go mod tidy` to add new dependencies

## Architecture Overview

### Provider Structure

- **Entry Point**: `internal/provider/provider.go` - Main provider implementation with configuration for API URL and token
- **Implemented Resources**: Currently supports seven resource types:
  - `panther_s3_source` - S3 log source integration (GraphQL-based)
  - `panther_httpsource` - HTTP log source integration (REST-based)
  - `panther_cloud_account` - AWS cloud account integration with exclusion features (GraphQL-based)
  - `panther_user` - User management and invitation (GraphQL-based)
  - `panther_role` - Role-based access control (GraphQL-based)
  - `panther_rule` - Detection rule management (REST-based with generated schema)
  - `panther_schema` - Custom log schema management (GraphQL-based)

### Client Architecture

The provider uses a dual-client approach within a unified `APIClient`:

- **GraphQL Client** (`internal/client/panther/panther.go`) - For S3 sources, cloud accounts, users, and roles
- **REST Client** (`internal/client/panther/panther.go`) - For HTTP sources and rules
- **Client Interface** (`internal/client/panther.go`) - Defines contracts for both client types
- **Unified API Client**: `APIClient` struct embeds both clients, accessible via `.GraphQLClient` and `.RestClient`

### Resource Implementation Patterns

**GraphQL Resources** (manually implemented):

- S3 source: Basic GraphQL CRUD with S3-specific schema
- Cloud account: GraphQL CRUD with complex exclusion lists (regions, resource types, regex patterns)
- User: GraphQL CRUD with role assignment and invitation flow
- Role: GraphQL CRUD with permissions management
- Schema: Uses GraphQL introspection for field discovery

**REST Resources** (code-generated with manual CRUD):

- HTTP source: Uses OpenAPI code generation with manual CRUD implementation
- Rule: Uses generated schema from OpenAPI with manual CRUD operations

### Code Generation Configuration

`generator_config.yml` defines REST endpoint mappings for multiple resource types:

- `httpsource` - Implemented (manual CRUD)
- `rule` - Implemented (uses generated schema)
- `policy`, `datamodel`, `global` - Scaffolded but not implemented
- Each resource configured with standard CRUD operations and ID field ignoring

### Authentication

Provider accepts authentication via:

- Environment variables: `PANTHER_API_URL` and `PANTHER_API_TOKEN`
- Provider configuration block in Terraform files
- Precedence: Provider config overrides environment variables

### Resource Implementation Notes

- **Sensitive Fields**: HTTP source contains sensitive auth fields that cannot be read after creation
- **Import Support**: All resources implement import using `ImportStatePassthroughID`
- **Generated Schema**: Rule resource uses generated schema with manual ID field addition and `UseStateForUnknown` configuration
- **State Consistency**: Cloud account handles null vs empty array states properly for optional list fields
- **GraphQL Introspection**: Use GraphQL introspection queries to discover exact field structures
- **URL Flexibility**: REST client handles both AWS API Gateway (`/v1` prefix) and direct Panther URLs

### Testing Requirements

- Acceptance tests require live Panther API credentials and create real resources
- Tests may incur costs in production environments  
- Use dummy data (like AWS account ID `999999999999`) for safe testing
- All tests follow `Test<Resource>Resource` naming convention
- Current test status: 8/8 tests passing
- Tests include comprehensive CRUD and import validation

### Test Coverage Status

- ✅ TestHttpSourceResource - HTTP source with deletion handling
- ✅ TestCloudAccountResource - Cloud account with state consistency
- ✅ TestRoleResource - GraphQL-based role management
- ✅ TestRuleResource - REST-based rule with generated schema
- ✅ TestS3SourceResource - S3 log source integration
- ✅ TestUserResource - GraphQL-based user management
- ✅ TestCreateAPIClient tests - Client functionality
- ✅ TestPrefixLogTypes tests - Helper functions

## Reference Links

- The Panther GraphQL Schema is at https://panther-community-us-east-1.s3.amazonaws.com/latest/graphql/schema.public.graphql