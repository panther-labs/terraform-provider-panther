terraform {
  required_version = ">= 1.0"
  required_providers {
    panther = {
      source = "panther-labs/panther"
      version = ">=0.2.4"
    }
  }
}

provider "panther" {
  token = var.token
  url   = var.url
}
