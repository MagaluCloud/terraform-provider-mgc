terraform {
  required_providers {
    mgc = {
      source = "magalucloud/mgc"
    }
  }
}

provider "mgc" {
  api_key = var.api_key
  region  = var.region
  key_pair_id = var.key_pair_id
  key_pair_secret = var.key_pair_secret
}

variable "api_key" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud API Key"
}

variable "region" {
  type        = string
  description = "The Magalu Cloud region"
}

variable "key_pair_id" {
  type        = string
  description = "The Magalu Cloud key pair id"
  sensitive   = true
}

variable "key_pair_secret" {
  type        = string
  description = "The Magalu Cloud key pair secret"
  sensitive   = true
}