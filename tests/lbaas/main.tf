# Generate random suffix for unique naming
resource "random_id" "suffix" {
  byte_length = 4
}

# Data sources for existing resources (replace with actual IDs)
# These would typically be data sources or variables in a real scenario
locals {
  vpc_id = "b44e3fe0-b609-4906-81a3-256e0cf68cfb"
}

# Scenario 1: Basic HTTP Load Balancer
resource "mgc_lbaas_network" "basic_http_lb" {
  name        = "basic-http-lb-${random_id.suffix.hex}-and-more-and-more"
  description = "Basic HTTP load balancer for web servers and more"
  type        = "proxy"
  visibility  = "internal"
  vpc_id      = local.vpc_id

  tls_certificates = [
    # {
    #   name        = "web-ssl-cert"
    #   description = "SSL certificate for web application"
    #   certificate = base64encode(file("${path.module}/certs/api_certificate.pem"))
    #   private_key = base64encode(file("${path.module}/certs/api_private_key.pem"))
    # }
  ]

  health_checks = [
    {
      name                      = "https-health-check"
      description               = "Health check for HTTPS endpoints"
      protocol                  = "http"
      port                      = 443
      path                      = "/health"
      healthy_status_code       = 200
      healthy_threshold_count   = 3
      unhealthy_threshold_count = 3
      interval_seconds          = 30
      timeout_seconds           = 10
      initial_delay_seconds     = 60
    }
  ]

  # Backend configuration for web servers
  backends = [
    {
      name              = "web-backend"
      description       = "Backend for web servers"
      balance_algorithm = "round_robin"
      targets_type      = "raw"
      panic_threshold   = 87
      health_check_name = "https-health-check"

      targets = [
        {
          ip_address = "192.168.30.4"
          port       = 8080
        },
        {
          ip_address = "192.168.30.4"
          port       = 8081
        }
      ]
    }
  ]

  # HTTP listener
  listeners = [
    {
      name                 = "http-listener"
      description          = "HTTP listener on port 80"
      port                 = 80
      protocol             = "tls"
      backend_name         = "web-backend"
      tls_certificate_name = "web-ssl-cert"
    }
  ]

  # Basic ACL to allow all HTTP traffic
  acls = [
    {
      name             = "allow-http"
      action           = "DENY"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    },
    {
      name             = "allow-http-2"
      action           = "DENY"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    },
    {
      name             = "allow-http-3"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]
}

data "mgc_lbaas_network" "basic_http_lb" {
  id = mgc_lbaas_network.basic_http_lb.id
}

output "scenario_1" {
  value = data.mgc_lbaas_network.basic_http_lb
}

data "mgc_lbaas_networks" "lbs" {
}

output "all_lbs" {
  value = data.mgc_lbaas_networks.lbs
}

data "mgc_lbaas_network_backend" "lbs_network_backend" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = element(tolist(mgc_lbaas_network.basic_http_lb.backends), 0).id
}

output "lbs_network_backend" {
  value = data.mgc_lbaas_network_backend.lbs_network_backend
}

data "mgc_lbaas_network_backends" "lbs_network_backends" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_backends" {
  value = data.mgc_lbaas_network_backends.lbs_network_backends
}

data "mgc_lbaas_network_listener" "lbs_network_listener" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = element(tolist(mgc_lbaas_network.basic_http_lb.listeners), 0).id
}

output "lbs_network_listener" {
  value = data.mgc_lbaas_network_listener.lbs_network_listener
}

data "mgc_lbaas_network_listeners" "lbs_network_listeners" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_listeners" {
  value = data.mgc_lbaas_network_listeners.lbs_network_listeners
}

data "mgc_lbaas_network_healthcheck" "lbs_network_healthcheck" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = element(tolist(mgc_lbaas_network.basic_http_lb.health_checks), 0).id
}

output "lbs_network_healthcheck" {
  value = data.mgc_lbaas_network_healthcheck.lbs_network_healthcheck
}

data "mgc_lbaas_network_healthchecks" "lbs_network_healthchecks" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_healthchecks" {
  value = data.mgc_lbaas_network_healthchecks.lbs_network_healthchecks
}

data "mgc_lbaas_network_certificate" "lbs_network_certificate" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = element(tolist(mgc_lbaas_network.basic_http_lb.tls_certificates), 0).id
}

output "lbs_network_certificate" {
  value = data.mgc_lbaas_network_certificate.lbs_network_certificate
}

data "mgc_lbaas_network_certificates" "lbs_network_certificates" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_certificates" {
  value = data.mgc_lbaas_network_certificates.lbs_network_certificates
}
