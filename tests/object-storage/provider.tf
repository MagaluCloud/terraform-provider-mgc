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

variable "mgc_access_key" {
  description = "Magalu Cloud access key."
  type        = string
  sensitive   = true
}

variable "mgc_secret_key" {
  description = "Magalu Cloud secret key."
  type        = string
  sensitive   = true
}

provider "mgc" {
  region     = var.mgc_region
  api_key    = var.mgc_api_key
  access_key = var.mgc_access_key
  secret_key = var.mgc_secret_key
}
