terraform {
  required_providers {
    mgc = {
      source = "magalucloud/mgc"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.1"
    }
  }
}

provider "mgc" {
  region  = var.region
  api_key = var.api_key
}

variable "region" {
  type    = string
  default = "br-ne1"
}
variable "api_key" {
  type      = string
  sensitive = true
}
