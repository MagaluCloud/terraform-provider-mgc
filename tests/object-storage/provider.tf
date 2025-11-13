terraform {
  required_providers {
    mgc = {
      source = "magalucloud/mgc"
    }
  }
}

variable "mgc_region" {
  description = "Region for Magalu Cloud object storage."
  type        = string
  default     = "br-ne1"
}

variable "mgc_api_key" {
  description = "Magalu Cloud API key."
  type        = string
  sensitive   = true
}

variable "mgc_key_pair_id" {
  description = "Magalu Cloud key pair id."
  type        = string
  sensitive   = true
}

variable "mgc_key_pair_secret" {
  description = "Magalu Cloud key pair secret."
  type        = string
  sensitive   = true
}

provider "mgc" {
  region     = var.mgc_region
  api_key    = var.mgc_api_key
  key_pair_id = var.mgc_key_pair_id
  key_pair_secret = var.mgc_key_pair_secret
}
