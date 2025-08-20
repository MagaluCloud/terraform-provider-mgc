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
  name        = "basic-http-lb-${random_id.suffix.hex}-and-more"
  description = "Basic HTTP load balancer for web servers"
  type        = "proxy"
  visibility  = "internal"
  vpc_id      = local.vpc_id

  # Backend configuration for web servers
  backends = [
    {
      name              = "web-backend"
      description       = "Backend for web servers"
      balance_algorithm = "round_robin"
      targets_type      = "raw"
      panic_threshold   = 50

      targets = [
        {
          ip_address = "192.168.30.4"
          port       = 80
        }
      ]
    }
  ]

  # HTTP listener
  listeners = [
    {
      name         = "http-listener"
      description  = "HTTP listener on port 80"
      port         = 80
      protocol     = "tcp"
      backend_name = "web-backend"
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
    }
  ]
}

# # Scenario 2: HTTPS Load Balancer with SSL Certificate
# resource "mgc_lbaas_network" "https_lb_with_ssl" {
#   name          = "https-lb-ssl-${random_id.suffix.hex}"
#   description   = "HTTPS load balancer with SSL termination"
#   public_ip_id  = local.public_ip_id
#   type          = "proxy"
#   visibility    = "external"
#   vpc_id        = local.vpc_id

#   # TLS Certificate configuration
# tls_certificates = [
#   {
#     name        = "web-ssl-cert"
#     description = "SSL certificate for web application"
#     certificate = file("${path.module}/certs/certificate.pem")
#     private_key = file("${path.module}/certs/private_key.pem")
#   }
#   ]

#   # Backend for HTTPS
#   backends = [
#     {
#       name                                     = "secure-web-backend"
#       description                              = "Secure backend for HTTPS traffic"
#       balance_algorithm                        = "round_robin"
#       targets_type                             = "instance"
#       health_check_name                        = "https-health-check"
#       close_connections_on_host_health_failure = true

#       targets = [
#         {
#           nic_id = local.instance_1_nic
#           port   = 443
#         },
#         {
#           nic_id = local.instance_2_nic
#           port   = 443
#         }
#       ]
#     }
#   ]

# # HTTPS health check
# health_checks = [
#   {
#     name                      = "https-health-check"
#     description               = "Health check for HTTPS endpoints"
#     protocol                  = "http"
#     port                      = 443
#     path                      = "/health"
#     healthy_status_code       = 200
#     healthy_threshold_count   = 3
#     unhealthy_threshold_count = 3
#     interval_seconds          = 30
#     timeout_seconds           = 10
#     initial_delay_seconds     = 60
#   }
# ]

#   # HTTPS listener with SSL certificate
#   listeners = [
#     {
#       name                 = "https-listener"
#       description          = "HTTPS listener with SSL termination"
#       port                 = 443
#       protocol             = "tls"
#       backend_name         = "secure-web-backend"
#       tls_certificate_name = "web-ssl-cert"
#     }
#   ]

#   # Security ACLs
#   acls = [
#     {
#       name             = "allow-https"
#       action           = "ALLOW"
#       ethertype        = "IPv4"
#       protocol         = "tcp"
#       remote_ip_prefix = "0.0.0.0/0"
#     },
#     {
#       name             = "deny-suspicious-ips"
#       action           = "DENY"
#       ethertype        = "IPv4"
#       protocol         = "tcp"
#       remote_ip_prefix = "192.168.100.0/24"
#     }
#   ]
# }

# # Scenario 3: Internal TCP Load Balancer for Database
# resource "mgc_lbaas_network" "internal_tcp_lb" {
#   name          = "internal-db-lb-${random_id.suffix.hex}"
#   description   = "Internal TCP load balancer for database cluster"
#   type          = "proxy"
#   visibility    = "internal"
#   vpc_id        = local.vpc_id

#   # Database backend
#   backends = [
#     {
#       name              = "database-backend"
#       description       = "Backend for database cluster"
#       balance_algorithm = "round_robin"
#       targets_type      = "instance"
#       health_check_name = "tcp-health-check"

#       targets = [
#         {
#           nic_id = local.instance_1_nic
#           port   = 5432
#         },
#         {
#           nic_id = local.instance_2_nic
#           port   = 5432
#         }
#       ]
#     }
#   ]

#   # TCP health check for database
#   health_checks = [
#     {
#       name                      = "tcp-health-check"
#       description               = "TCP health check for database"
#       protocol                  = "tcp"
#       port                      = 5432
#       healthy_threshold_count   = 2
#       unhealthy_threshold_count = 2
#       interval_seconds          = 15
#       timeout_seconds           = 5
#     }
#   ]

#   # TCP listener for database
#   listeners = [
#     {
#       name         = "database-listener"
#       description  = "TCP listener for database connections"
#       port         = 5432
#       protocol     = "tcp"
#       backend_name = "database-backend"
#     }
#   ]

#   # Internal network ACL
#   acls = [
#     {
#       name             = "allow-internal-db"
#       action           = "ALLOW"
#       ethertype        = "IPv4"
#       protocol         = "tcp"
#       remote_ip_prefix = "10.0.0.0/8"
#     }
#   ]
# }

# # Scenario 4: Multi-service Load Balancer (HTTP + HTTPS)
# resource "mgc_lbaas_network" "multi_service_lb" {
#   name          = "multi-service-lb-${random_id.suffix.hex}"
#   description   = "Load balancer handling multiple services and protocols"
#   public_ip_id  = local.public_ip_id
#   type          = "proxy"
#   visibility    = "external"
#   vpc_id        = local.vpc_id

#   # TLS Certificate for HTTPS
#   tls_certificates = [
#     {
#       name        = "api-ssl-cert"
#       description = "SSL certificate for API endpoints"
#       certificate = file("${path.module}/certs/api_certificate.pem")
#       private_key = file("${path.module}/certs/api_private_key.pem")
#     }
#   ]

#   # Backend for web application
#   backends = [
#     {
#       name              = "web-app-backend"
#       description       = "Backend for web application"
#       balance_algorithm = "round_robin"
#       targets_type      = "instance"
#       health_check_name = "web-health-check"

#       targets = [
#         {
#           nic_id = local.instance_1_nic
#           port   = 8080
#         },
#         {
#           nic_id = local.instance_2_nic
#           port   = 8080
#         }
#       ]
#     },
#     # Backend for API services
#     {
#       name              = "api-backend"
#       description       = "Backend for API services"
#       balance_algorithm = "round_robin"
#       targets_type      = "instance"
#       health_check_name = "api-health-check"

#       targets = [
#         {
#           nic_id = local.instance_1_nic
#           port   = 3000
#         },
#         {
#           nic_id = local.instance_2_nic
#           port   = 3000
#         }
#       ]
#     }
#   ]

#   # Health check for web application
#   health_checks = [
#     {
#       name                      = "web-health-check"
#       description               = "Health check for web application"
#       protocol                  = "http"
#       port                      = 8080
#       path                      = "/status"
#       healthy_status_code       = 200
#       healthy_threshold_count   = 2
#       unhealthy_threshold_count = 3
#       interval_seconds          = 30
#       timeout_seconds           = 10
#     },
#     # Health check for API
#     {
#       name                      = "api-health-check"
#       description               = "Health check for API services"
#       protocol                  = "http"
#       port                      = 3000
#       path                      = "/api/health"
#       healthy_status_code       = 200
#       healthy_threshold_count   = 2
#       unhealthy_threshold_count = 3
#       interval_seconds          = 20
#       timeout_seconds           = 8
#     }
#   ]

#   # HTTP listener for web application
#   listeners = [
#     {
#       name         = "web-http-listener"
#       description  = "HTTP listener for web application"
#       port         = 80
#       protocol     = "tcp"
#       backend_name = "web-app-backend"
#     },
#     # HTTPS listener for API
#     {
#       name                 = "api-https-listener"
#       description          = "HTTPS listener for API services"
#       port                 = 443
#       protocol             = "tls"
#       backend_name         = "api-backend"
#       tls_certificate_name = "api-ssl-cert"
#     }
#   ]

#   # ACL configuration
#   acls = [
#     {
#       name             = "allow-web-traffic"
#       action           = "ALLOW"
#       ethertype        = "IPv4"
#       protocol         = "tcp"
#       remote_ip_prefix = "0.0.0.0/0"
#     },
#     {
#       name             = "allow-api-traffic"
#       action           = "ALLOW"
#       ethertype        = "IPv4"
#       protocol         = "tcp"
#       remote_ip_prefix = "0.0.0.0/0"
#     },
#     {
#       name             = "deny-blocked-countries"
#       action           = "DENY"
#       ethertype        = "IPv4"
#       protocol         = "tcp"
#       remote_ip_prefix = "203.0.113.0/24"
#     }
#   ]
# }

# # Scenario 5: TCP Load Balancer for Gaming (UDP not supported)
# resource "mgc_lbaas_network" "tcp_gaming_lb" {
#   name          = "tcp-gaming-lb-${random_id.suffix.hex}"
#   description   = "TCP load balancer for gaming servers"
#   public_ip_id  = local.public_ip_id
#   type          = "proxy"
#   visibility    = "external"
#   vpc_id        = local.vpc_id

#   # Gaming server backend
#   backends = [
#     {
#       name              = "gaming-backend"
#       description       = "Backend for gaming servers"
#       balance_algorithm = "round_robin"
#       targets_type      = "instance"
#       health_check_name = "tcp-health-check-gaming"

#       targets = [
#         {
#           nic_id = local.instance_1_nic
#           port   = 7777
#         },
#         {
#           nic_id = local.instance_2_nic
#           port   = 7777
#         }
#       ]
#     }
#   ]

#   # TCP health check
#   health_checks = [
#     {
#       name                      = "tcp-health-check-gaming"
#       description               = "TCP health check for gaming servers"
#       protocol                  = "tcp"
#       port                      = 7777
#       healthy_threshold_count   = 2
#       unhealthy_threshold_count = 3
#       interval_seconds          = 20
#       timeout_seconds           = 5
#     }
#   ]

#   # TCP listener for gaming
#   listeners = [
#     {
#       name         = "gaming-tcp-listener"
#       description  = "TCP listener for gaming traffic"
#       port         = 7777
#       protocol     = "tcp"
#       backend_name = "gaming-backend"
#     }
#   ]

#   # ACL for gaming traffic
#   acls = [
#     {
#       name             = "allow-gaming-tcp"
#       action           = "ALLOW"
#       ethertype        = "IPv4"
#       protocol         = "tcp"
#       remote_ip_prefix = "0.0.0.0/0"
#     }
#   ]
# }

# # Output examples
# output "basic_http_lb_id" {
#   description = "ID of the basic HTTP load balancer"
#   value       = mgc_lbaas_network.basic_http_lb.id
# }

# output "https_lb_with_ssl_id" {
#   description = "ID of the HTTPS load balancer with SSL"
#   value       = mgc_lbaas_network.https_lb_with_ssl.id
# }

# output "internal_tcp_lb_id" {
#   description = "ID of the internal TCP load balancer"
#   value       = mgc_lbaas_network.internal_tcp_lb.id
# }

# output "multi_service_lb_id" {
#   description = "ID of the multi-service load balancer"
#   value       = mgc_lbaas_network.multi_service_lb.id
# }

# output "tcp_gaming_lb_id" {
#   description = "ID of the TCP gaming load balancer"
#   value       = mgc_lbaas_network.tcp_gaming_lb.id
# }
