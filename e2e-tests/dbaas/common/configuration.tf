variable "mgc_zone" {
  description = "AZDC Zone"
  type        = string
  default     = "a"
}

variable "mgc_region" {
  description = "MGC Region"
  type        = string
  default     = "br-ne1"
}

variable "env" {
  description = "Running Environment"
  type = string
  default = "pre-prod" # or "prod"
}

variable "api_key" {
  description = "API Key"
  type = string
  default = ""
}

provider "mgc" {
  region  = var.mgc_region
  api_key = var.api_key
  env     = var.env
}

variable "engine_name" {
  description = "Engine name"
  type = string
  default = "mysql"
}

variable "engine_version" {
  description = "Engine version"
  type = string
  default = "8.0"
}

variable "db_prefix" {
  description = "Prefix to create database instances"
  type = string
  default = "<DB_PREFIX>"
}

variable "db_password" {
  description = "Database password"
  type = string
  default = "<PASSWORD>"
}