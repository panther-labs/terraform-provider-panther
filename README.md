_This repo is under active development and is not recommended for production use_

# Panther Terraform Provider
Terraform provider for [Panther](https://panther.com/) resources


_This template repository is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). The template repository built on the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) can be found at [terraform-provider-scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding). See [Which SDK Should I Use?](https://www.terraform.io/docs/plugin/which-sdk.html) in the Terraform documentation for additional information._

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

## Usage

Use the examples directory and the corresponding [README.md](./examples/README.md) as a guide on setting up the provider
and trying out terraform command to create/update/delete resources.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

### Code generation

Starting with the `httpsource` resource, the resource scaffolding and schema are generated using the terraform
[framework code generator](https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator)
and the [openapi generator](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)
plugins. In order to update or create new resources, you need to install both these plugins as described in the links.

### Creating a new resource

In order to create a new resource in the Panther provider, it must already exist in the Panther REST API and provide
CRUD REST methods. The following steps are required to create a new resource:

1. Scaffold a new resource by running the following command:
```
   tfplugingen-framework scaffold resource \
   --name {resource_name}} \
   --output-dir ./internal/provider
```
2. Update the `generator_config.yml` file with the paths of the REST methods for the new resource.
3. Get the latest Panther OpenAPI schema locally and run the following command to update the `provider-code-specs.json`
specification file:
```
tfplugingen-openapi generate \
  --config ./generator_config.yml \
  --output ./provider-code-spec.json \
    {path_to_openapi_yml}
```
4. Run the following command to populate the resource model/schema:
```
tfplugingen-framework generate resources \
  --input ./provider-code-spec.json \
  --output ./internal/provider
```
5. Implement the CRUD methods in the resource file under `internal/provider/{resource_name}_resource.go` as is done in
the `httpsource` resource. If creating a new resource that requires `Resource Import` functionality, you have to add the
you have to implement the `ImportState` method in the resource file, as is done in the `httpsource` resource.

### Updating an existing resource

In order to update an existing resource, e.g. because of a schema change or to add new attributes, perform the following
steps:

1. Make sure the `generator_config.yml` file is up to date. This has to be changed only for updates to existing 
REST endpoints/resources.
2. Follow steps `3` and `4` from the `Creating a new resource` section.

### Code generation limitations

The code generation tools currently do not cover all the functionality we need. For this reason, setting the defaults for
optional values and setting the `UseStateForUnknownn` value for the `id` in the schema is done manually in the resource
`Schema` method. Additionally, as mentioned above, there is no support for importing the state of a resource, so the
`ImportState` method has to be implemented manually.

### Testing

```shell
In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources and may cost money to run.

```shell
PANTHER_API_URL=<Panther environment URL> \
PANTHER_API_TOKEN=<Panther API Token> \
make testacc
```

In order to manually test the provider refer to the [Usage](#usage) section above.

### Import limitations

The http source resource contains sensitive values for `security_password` and `security_secret_value`, which cannot be read after
being created. For this reason, make sure to avoid updating these in the console as they cannot be reflected to the state of the resource
in Terraform. This applies to importing the state of the resource as well from an existing resource. If updating these values
from the console or importing an existing resource, you will need to run `terraform apply` with the appropriate values to reflect
the changes in the state of the resource.