terraform {
  required_version = ">= 1.6.0"

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "~> 2.64"
    }
  }
}

provider "scaleway" {
  project_id      = var.project_id
  organization_id = var.organization_id
  region          = var.region
  zone            = var.zone
}

variable "project_id" {
  type        = string
  default     = null
  nullable    = true
  description = "Optional. If null, provider default project resolution is used."
}

variable "organization_id" {
  type        = string
  default     = null
  nullable    = true
  description = "Optional. If null, provider default organization resolution is used."
}

variable "region" {
  type    = string
  default = "fr-par"
}

variable "zone" {
  type    = string
  default = "fr-par-1"
}

variable "web_type" {
  type    = string
  default = "DEV1-M"
}

variable "web_replicas" {
  type    = number
  default = 1
}

variable "block_size_gb" {
  type    = number
  default = 500
}

variable "load_balancer_type" {
  type    = string
  default = "LB-S"
}

variable "rdb_node_type" {
  type    = string
  default = "DB-DEV-S"
}

variable "redis_node_type" {
  type    = string
  default = "RED1-MICRO"
}

variable "redis_cluster_size" {
  type    = number
  default = 3
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "redis_password" {
  type      = string
  sensitive = true
}

resource "scaleway_block_volume" "data" {
  name       = "impact-showcase-block"
  size_in_gb = var.block_size_gb
  iops       = 5000
  zone       = var.zone
}

resource "scaleway_instance_server" "web" {
  count = var.web_replicas

  name  = "impact-showcase-web-${count.index + 1}"
  type  = var.web_type
  image = "ubuntu_jammy"
  zone  = var.zone
}

resource "scaleway_lb" "edge" {
  name = "impact-showcase-lb"
  type = var.load_balancer_type
  zone = var.zone
}

resource "scaleway_rdb_instance" "db" {
  name           = "impact-showcase-rdb"
  node_type      = var.rdb_node_type
  engine         = "PostgreSQL-15"
  is_ha_cluster  = false
  disable_backup = true
  user_name      = "impact"
  password       = var.db_password
  region         = var.region
}

resource "scaleway_redis_cluster" "cache" {
  name         = "impact-showcase-redis"
  version      = "6.2.7"
  node_type    = var.redis_node_type
  user_name    = "impact"
  password     = var.redis_password
  cluster_size = var.redis_cluster_size
  tls_enabled  = true
  zone         = var.zone

  acl {
    ip          = "0.0.0.0/0"
    description = "showcase-only"
  }
}

# intentionally unsupported in impact mapping to showcase diagnostics output
resource "scaleway_instance_ip" "unsupported_web_ip" {}

# intentionally unsupported in impact mapping to showcase diagnostics output
resource "scaleway_vpc_private_network" "unsupported_private_network" {
  name   = "impact-showcase-pn"
  region = var.region
}
