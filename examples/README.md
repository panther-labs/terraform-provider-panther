# Examples

This directory contains examples that can be used to practically try out or test the provider and the resources it contains.
In order to do that you have to:

1. Build the provider as described in the main [README.md](../README.md)
2. Add a `.terraformrc` file in your home dir which contains the following:
```hcl
provider_installation {
  dev_overrides {
    "panther-labs/panther" = "{PATH}"
  }
  direct {}
}
```
where `PATH` is the path that your go binaries are located. This will either be your `GOBIN` var if it's set, or `{GOPATH}/bin`.
3. Navigate to the example directory you want to try out. The examples are set up so that a `variables.tf` file is created which
includes the provider level variables you need to run terraform commands (`token`, `url`). The easiest way to use these
examples is to create a `*.tfvars` file to get the values for these variables, like in the example below:
```terraform
token                 = "{your_token}"
url                   = "{your_url}"
integration_label     = "test-label"
log_stream_type       = "Auto"
log_types             = ["AWS.CloudFrontAccess"]
security_type         = "SharedSecret"
security_header_key   = "x-api-key"
security_secret_value = "test-secret"
```
That way you can run terraform commands like
```shell
terraform plan -var-file="your-file.tfvars"
```
and
```shell
terraform apply -var-file="your-file.tfvars"
```
or alternatively you can provide the variables directly in the command line:
```shell
terraform plan -var="var1=value1" -var="var2=value2" ...
```

This will create actual resources in your dev environment, so make sure to run
```shell
terraform destroy -var-file="your-file.tfvars"
```
when you are done with testing, so no lingering resources are left.

Don't forget to remove the `.terraformrc` file when you are done testing and if you plan to use the released provider version. 



The document generation tool looks for files in the following locations by default. All other *.tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating examples that can run and/or ar testable even if some parts are not relevant for the documentation.

* **provider/provider.tf** example file for the provider index page
* **data-sources/`full data source name`/data-source.tf** example file for the named data source page
* **resources/`full resource name`/resource.tf** example file for the named data source page
