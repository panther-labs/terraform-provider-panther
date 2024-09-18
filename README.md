_This repo is under active development and is not recommended for production use_

# Panther Terraform Provider
Terraform provider for [Panther](https://panther.com/) resources


_This template repository is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). The template repository built on the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) can be found at [terraform-provider-scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding). See [Which SDK Should I Use?](https://www.terraform.io/docs/plugin/which-sdk.html) in the Terraform documentation for additional information._

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

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

Use the examples directory as a guide for setting up the provider.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources and may cost money to run.

```shell
PANTHER_API_URL=<Panther GraphQL URL> \
PANTHER_API_TOKEN=<Panther GraphQL API Token> \
make testacc
```

There are also complete examples under the `examples` directory. If you want to try out the provider as you are building
it, you can add a `.terraformrc` file in your home dir which contains the following:
```hcl
provider_installation {
  dev_overrides {
    "panther-labs/panther" = "{PATH}"
  }
  direct {}
}
```
where `PATH` is the path that your go binaries are. This will either be your `GOBIN` var if it's set, or `{GOPATH}/bin`.

Then you can normally run terraform commands to create the resources, like
```shell
terraform plan -var="var1=value1" -var="var2=value2" ...
```
or by adding variables to a temporary `.tfvars` file.
As this will create actual resources in your dev environment, make sure to run 
```shell
terraform destroy
```
when you are done with testing, so no lingering resources are left.
