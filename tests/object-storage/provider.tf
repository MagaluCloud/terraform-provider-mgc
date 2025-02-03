terraform {
  required_providers {
    mgc = {
      source = "magalucloud/mgc"
    }
  }
}

provider "mgc" {
  region  = var.region
  api_key = var.api_key

  object_storage = {
    key_pair = {
      key_id     = var.key_id
      key_secret = var.key_secret
    }
  }
}

variable "region" {
  type    = string
  default = "br-ne1"
}
variable "api_key" {
  type      = string
  sensitive = true
}

variable "key_id" {
  type      = string
  sensitive = true
}

variable "key_secret" {
  type      = string
  sensitive = true
}
