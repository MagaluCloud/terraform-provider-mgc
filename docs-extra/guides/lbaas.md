---
page_title: "Load Balancer as a Service (LBaaS) in Magalu Cloud"
subcategory: "Guides"
description: "A comprehensive guide to creating and managing Load Balancers in Magalu Cloud, including backends, listeners, health checks, SSL/TLS termination, and security configurations."
---

# Comprehensive Guide to Load Balancer as a Service (LBaaS) in Magalu Cloud

This guide will help you understand and implement Load Balancers in Magalu Cloud, explaining how to distribute traffic across multiple targets, configure SSL/TLS termination, implement health checks, and manage security policies.

## Core LBaaS Concepts in Magalu Cloud

Magalu Cloud's Load Balancer service provides a highly available and scalable solution for distributing incoming traffic across multiple targets. The LBaaS architecture consists of several interconnected components:

1. **Load Balancer**: The main entry point that receives and distributes traffic
2. **Listeners**: Define the ports and protocols the load balancer accepts traffic on
3. **Backends**: Groups of targets that receive the distributed traffic
4. **Targets**: The actual servers or instances that handle requests
5. **Health Checks**: Monitor target health and remove unhealthy targets from rotation
6. **TLS Certificates**: Enable SSL/TLS termination for secure connections
7. **ACLs (Access Control Lists)**: Control which traffic is allowed or denied

## Understanding Load Balancer Types and Visibility

### Load Balancer Types

Currently, Magalu Cloud supports:

- **Proxy Type**: Layer 7 load balancer with advanced traffic management capabilities

### Visibility Options

- **Internal**: Accessible only within the VPC (private load balancer)
- **External**: Accessible from the internet (requires a public IP)

## Basic Load Balancer Setup

### Step 1: Create a Simple HTTP Load Balancer

Let's start with a basic internal load balancer:

```terraform
resource "mgc_lbaas_network" "web_lb" {
  name        = "web-load-balancer"
  description = "Load balancer for web servers"
  type        = "proxy"
  visibility  = "internal"
  vpc_id      = "your-vpc-id"

  # Backend configuration
  backends = [
    {
      name              = "web-backend"
      description       = "Backend for web servers"
      balance_algorithm = "round_robin"
      targets_type      = "raw"  # Use IP addresses directly

      targets = [
        {
          ip_address = "10.0.1.10"
          port       = 80
        },
        {
          ip_address = "10.0.1.11"
          port       = 80
        }
      ]
    }
  ]

  # Listener configuration
  listeners = [
    {
      name         = "http-listener"
      description  = "HTTP traffic on port 80"
      port         = 80
      protocol     = "tcp"
      backend_name = "web-backend"  # References the backend by name
    }
  ]

  # Basic ACL to allow HTTP traffic
  acls = [
    {
      name             = "allow-http"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]
}
```

## Creating External Load Balancers

For external load balancers, you need a public IP:

```terraform
# First, create a public IP
resource "mgc_network_public_ips" "lb_public_ip" {
  description = "Public IP for load balancer"
  vpc_id      = "your-vpc-id"
}

# Then create the external load balancer
resource "mgc_lbaas_network" "external_lb" {
  name         = "external-web-lb"
  description  = "External load balancer for web services"
  type         = "proxy"
  visibility   = "external"
  vpc_id       = "your-vpc-id"
  public_ip_id = mgc_network_public_ips.lb_public_ip.id

  backends = [
    {
      name              = "external-backend"
      description       = "Backend for external web services"
      balance_algorithm = "round_robin"
      targets_type      = "raw"

      targets = [
        {
          ip_address = "10.0.1.10"
          port       = 80
        },
        {
          ip_address = "10.0.1.11"
          port       = 80
        }
      ]
    }
  ]

  listeners = [
    {
      name         = "external-http-listener"
      description  = "External HTTP listener"
      port         = 80
      protocol     = "tcp"
      backend_name = "external-backend"
    }
  ]

  acls = [
    {
      name             = "allow-external-http"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]
}
```

## Target Types: Raw IP vs Instance NICs

### Using Raw IP Addresses

When you know the exact IP addresses of your targets:

```terraform
backends = [
  {
    name              = "raw-backend"
    balance_algorithm = "round_robin"
    targets_type      = "raw"

    targets = [
      {
        ip_address = "192.168.1.10"
        port       = 8080
      },
      {
        ip_address = "192.168.1.11"
        port       = 8080
      }
    ]
  }
]
```

### Using Instance Network Interfaces

When targeting VM instances via their network interface IDs:

```terraform
# First, get the network interface IDs from your VMs
locals {
  vm_interface_ids = [
    for interface in mgc_virtual_machine_instances.web_server.network_interfaces :
    interface.id if interface.primary
  ]
}

backends = [
  {
    name              = "instance-backend"
    balance_algorithm = "round_robin"
    targets_type      = "instance"

    targets = [
      {
        nic_id = local.vm_interface_ids[0]
        port   = 8080
      }
    ]
  }
]
```

## Implementing Health Checks

Health checks ensure traffic is only sent to healthy targets:

```terraform
resource "mgc_lbaas_network" "lb_with_health_checks" {
  name        = "lb-with-health-checks"
  description = "Load balancer with comprehensive health monitoring"
  type        = "proxy"
  visibility  = "internal"
  vpc_id      = "your-vpc-id"

  # Define health checks
  health_checks = [
    {
      name                      = "web-health-check"
      description               = "HTTP health check for web servers"
      protocol                  = "http"
      port                      = 80
      path                      = "/health"           # Health check endpoint
      healthy_status_code       = 200                # Expected HTTP status
      healthy_threshold_count   = 3                  # Consecutive successes needed
      unhealthy_threshold_count = 2                  # Consecutive failures before marking unhealthy
      interval_seconds          = 30                 # Check every 30 seconds
      timeout_seconds           = 10                 # 10 second timeout
      initial_delay_seconds     = 60                 # Wait 60s before first check
    },
    {
      name                      = "tcp-health-check"
      description               = "TCP health check for database"
      protocol                  = "tcp"
      port                      = 5432
      healthy_threshold_count   = 2
      unhealthy_threshold_count = 3
      interval_seconds          = 20
      timeout_seconds           = 5
    }
  ]

  backends = [
    {
      name              = "web-backend-with-health"
      description       = "Web backend with health monitoring"
      balance_algorithm = "round_robin"
      targets_type      = "raw"
      health_check_name = "web-health-check"  # Reference health check by name
      panic_threshold   = 50                  # Panic mode when 50% targets unhealthy

      targets = [
        {
          ip_address = "10.0.1.10"
          port       = 80
        },
        {
          ip_address = "10.0.1.11"
          port       = 80
        }
      ]
    }
  ]

  listeners = [
    {
      name         = "web-listener"
      port         = 80
      protocol     = "tcp"
      backend_name = "web-backend-with-health"
    }
  ]

  acls = [
    {
      name             = "allow-web-traffic"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "10.0.0.0/8"
    }
  ]
}
```

## SSL/TLS Termination

Configure HTTPS termination with TLS certificates:

```terraform
resource "mgc_lbaas_network" "https_lb" {
  name        = "https-load-balancer"
  description = "HTTPS load balancer with SSL termination"
  type        = "proxy"
  visibility  = "external"
  vpc_id      = "your-vpc-id"
  public_ip_id = mgc_network_public_ips.lb_public_ip.id

  # TLS certificate configuration
  tls_certificates = [
    {
      name        = "web-ssl-cert"
      description = "SSL certificate for web application"
      certificate = base64encode(file("path/to/certificate.pem"))
      private_key = base64encode(file("path/to/private-key.pem"))
    }
  ]

  backends = [
    {
      name              = "https-backend"
      description       = "Backend for HTTPS traffic"
      balance_algorithm = "round_robin"
      targets_type      = "raw"

      targets = [
        {
          ip_address = "10.0.1.10"
          port       = 80  # Backend can still use HTTP
        },
        {
          ip_address = "10.0.1.11"
          port       = 80
        }
      ]
    }
  ]

  listeners = [
    {
      name                 = "https-listener"
      description          = "HTTPS listener with SSL termination"
      port                 = 443
      protocol             = "tls"               # TLS protocol for HTTPS
      backend_name         = "https-backend"
      tls_certificate_name = "web-ssl-cert"     # Reference certificate by name
    }
  ]

  acls = [
    {
      name             = "allow-https"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]
}
```

## Advanced ACL Configuration

Implement sophisticated access control policies:

```terraform
resource "mgc_lbaas_network" "secure_lb" {
  name        = "secure-load-balancer"
  description = "Load balancer with advanced security policies"
  type        = "proxy"
  visibility  = "external"
  vpc_id      = "your-vpc-id"
  public_ip_id = mgc_network_public_ips.lb_public_ip.id

  backends = [
    {
      name              = "secure-backend"
      balance_algorithm = "round_robin"
      targets_type      = "raw"

      targets = [
        {
          ip_address = "10.0.1.10"
          port       = 443
        }
      ]
    }
  ]

  listeners = [
    {
      name         = "secure-listener"
      port         = 443
      protocol     = "tcp"
      backend_name = "secure-backend"
    }
  ]

  # Multiple ACL rules - processed in order
  acls = [
    # Deny specific malicious IP range
    {
      name             = "deny-malicious-ips"
      action           = "DENY"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "192.0.2.0/24"  # Example malicious range
    },
    # Allow from corporate network
    {
      name             = "allow-corporate"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "203.0.113.0/24"  # Corporate IP range
    },
    # Allow from partner networks
    {
      name             = "allow-partners"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "198.51.100.0/24"  # Partner IP range
    },
    # Default deny for unspecified traffic
    {
      name             = "default-deny"
      action           = "DENY_UNSPECIFIED"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]
}
```

## Complete Multi-Tier Application Example

Here's a comprehensive example showing a complete web application architecture:

```terraform
# Network resources
resource "mgc_network_vpcs" "app_vpc" {
  name        = "application-vpc"
  description = "VPC for multi-tier application"
}

resource "mgc_network_subnetpools" "app_pool" {
  name = "app-subnet-pool"
  cidr = "10.0.0.0/16"
}

resource "mgc_network_vpcs_subnets" "web_subnet" {
  name            = "web-subnet"
  vpc_id          = mgc_network_vpcs.app_vpc.id
  subnetpool_id   = mgc_network_subnetpools.app_pool.id
  cidr_block      = "10.0.1.0/24"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
}

resource "mgc_network_vpcs_subnets" "app_subnet" {
  name            = "app-subnet"
  vpc_id          = mgc_network_vpcs.app_vpc.id
  subnetpool_id   = mgc_network_subnetpools.app_pool.id
  cidr_block      = "10.0.2.0/24"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
}

# Public IP for external load balancer
resource "mgc_network_public_ips" "app_lb_ip" {
  description = "Public IP for application load balancer"
  vpc_id      = mgc_network_vpcs.app_vpc.id
}

# External load balancer for web tier
resource "mgc_lbaas_network" "web_lb" {
  name         = "web-load-balancer"
  description  = "External load balancer for web tier"
  type         = "proxy"
  visibility   = "external"
  vpc_id       = mgc_network_vpcs.app_vpc.id
  public_ip_id = mgc_network_public_ips.app_lb_ip.id

  # SSL certificate for HTTPS
  tls_certificates = [
    {
      name        = "web-ssl-cert"
      description = "SSL certificate for web application"
      certificate = base64encode(file("certs/web-cert.pem"))
      private_key = base64encode(file("certs/web-key.pem"))
    }
  ]

  # Health check for web servers
  health_checks = [
    {
      name                      = "web-health"
      description               = "Health check for web servers"
      protocol                  = "http"
      port                      = 80
      path                      = "/health"
      healthy_status_code       = 200
      healthy_threshold_count   = 2
      unhealthy_threshold_count = 3
      interval_seconds          = 30
      timeout_seconds           = 10
    }
  ]

  # Web backend
  backends = [
    {
      name              = "web-backend"
      description       = "Backend for web servers"
      balance_algorithm = "round_robin"
      targets_type      = "raw"
      health_check_name = "web-health"
      panic_threshold   = 75

      targets = [
        {
          ip_address = "10.0.1.10"
          port       = 80
        },
        {
          ip_address = "10.0.1.11"
          port       = 80
        },
        {
          ip_address = "10.0.1.12"
          port       = 80
        }
      ]
    }
  ]

  # HTTPS listener
  listeners = [
    {
      name                 = "https-listener"
      description          = "HTTPS listener for web traffic"
      port                 = 443
      protocol             = "tls"
      backend_name         = "web-backend"
      tls_certificate_name = "web-ssl-cert"
    },
    # Optional HTTP listener for redirect
    {
      name         = "http-redirect"
      description  = "HTTP listener for redirect to HTTPS"
      port         = 80
      protocol     = "tcp"
      backend_name = "web-backend"
    }
  ]

  # Security ACLs
  acls = [
    {
      name             = "allow-https"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    },
    {
      name             = "allow-http"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]
}

# Internal load balancer for application tier
resource "mgc_lbaas_network" "app_lb" {
  name        = "app-load-balancer"
  description = "Internal load balancer for application tier"
  type        = "proxy"
  visibility  = "internal"
  vpc_id      = mgc_network_vpcs.app_vpc.id

  # Health check for application servers
  health_checks = [
    {
      name                      = "app-health"
      description               = "Health check for application servers"
      protocol                  = "http"
      port                      = 8080
      path                      = "/api/health"
      healthy_status_code       = 200
      healthy_threshold_count   = 2
      unhealthy_threshold_count = 2
      interval_seconds          = 20
      timeout_seconds           = 5
    }
  ]

  # Application backend
  backends = [
    {
      name              = "app-backend"
      description       = "Backend for application servers"
      balance_algorithm = "round_robin"
      targets_type      = "raw"
      health_check_name = "app-health"

      targets = [
        {
          ip_address = "10.0.2.10"
          port       = 8080
        },
        {
          ip_address = "10.0.2.11"
          port       = 8080
        }
      ]
    }
  ]

  # Application listener
  listeners = [
    {
      name         = "app-listener"
      description  = "Listener for application traffic"
      port         = 8080
      protocol     = "tcp"
      backend_name = "app-backend"
    }
  ]

  # ACL to allow traffic only from web tier
  acls = [
    {
      name             = "allow-from-web-tier"
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "10.0.1.0/24"  # Web subnet CIDR
    }
  ]
}

# Outputs
output "web_lb_public_ip" {
  description = "Public IP of the web load balancer"
  value       = mgc_network_public_ips.app_lb_ip.public_ip
}

output "web_lb_id" {
  description = "ID of the web load balancer"
  value       = mgc_lbaas_network.web_lb.id
}

output "app_lb_id" {
  description = "ID of the application load balancer"
  value       = mgc_lbaas_network.app_lb.id
}
```

## Understanding Resource References and Relationships

### How Components Reference Each Other

LBaaS resources use **name-based references** to connect components:

1. **Listeners reference Backends**: Use `backend_name` to specify which backend handles the traffic
2. **Listeners reference TLS Certificates**: Use `tls_certificate_name` for SSL termination
3. **Backends reference Health Checks**: Use `health_check_name` to associate health monitoring
4. **Targets reference Network Interfaces**: Use `nic_id` when `targets_type = "instance"`

### Reference Examples

```terraform
# Backend references health check by name
backends = [
  {
    name              = "my-backend"
    health_check_name = "my-health-check"  # Must match health check name
    # ... other configuration
  }
]

# Listener references backend by name
listeners = [
  {
    name         = "my-listener"
    backend_name = "my-backend"            # Must match backend name
    # ... other configuration
  }
]

# TLS listener references certificate by name
listeners = [
  {
    name                 = "https-listener"
    protocol             = "tls"
    tls_certificate_name = "my-ssl-cert"   # Must match certificate name
    # ... other configuration
  }
]
```

## What Can Be Updated vs What Requires Replacement

### Updatable Properties

These can be modified without recreating the load balancer:

- Load balancer `name` and `description`
- Backend `panic_threshold`
- Backend `targets` (add/remove/modify targets)
- Health check timing parameters (`interval_seconds`, `timeout_seconds`, etc.)
- Health check `path` and `healthy_status_code`
- ACL rules (entire ACL set can be replaced)

### Properties That Require Replacement

These require destroying and recreating the resource:

- Load balancer `type`, `visibility`, `vpc_id`, `public_ip_id`
- Backend `name`, `description`, `balance_algorithm`, `targets_type`, `health_check_name`
- Listener `name`, `port`, `protocol`, `backend_name`, `tls_certificate_name`
- Health check `name`, `protocol`, `port`
- TLS certificate `name`, `certificate`, `private_key`

### Update Example

```terraform
# This will update existing targets without replacement
resource "mgc_lbaas_network" "updateable_lb" {
  name = "my-load-balancer"
  # ... other configuration

  backends = [
    {
      name              = "my-backend"
      balance_algorithm = "round_robin"  # Cannot change without replacement
      targets_type      = "raw"          # Cannot change without replacement
      panic_threshold   = 60             # Can be updated

      # These targets can be added/removed/modified
      targets = [
        {
          ip_address = "10.0.1.10"
          port       = 80
        },
        {
          ip_address = "10.0.1.11"  # New target - will be added
          port       = 80
        }
        # Removed target will be automatically removed
      ]
    }
  ]
}
```

## Using Data Sources for Load Balancer Information

Query existing load balancer information:

```terraform
# Get information about a specific load balancer
data "mgc_lbaas_network" "existing_lb" {
  id = "your-load-balancer-id"
}

# List all load balancers
data "mgc_lbaas_networks" "all_lbs" {}

# Get specific backend information
data "mgc_lbaas_network_backend" "backend_info" {
  lb_id = data.mgc_lbaas_network.existing_lb.id
  id    = "backend-id"
}

# List all backends for a load balancer
data "mgc_lbaas_network_backends" "all_backends" {
  lb_id = data.mgc_lbaas_network.existing_lb.id
}

# Get listener information
data "mgc_lbaas_network_listeners" "all_listeners" {
  lb_id = data.mgc_lbaas_network.existing_lb.id
}

# Get health check information
data "mgc_lbaas_network_healthchecks" "all_health_checks" {
  lb_id = data.mgc_lbaas_network.existing_lb.id
}

# Get certificate information
data "mgc_lbaas_network_certificates" "all_certificates" {
  lb_id = data.mgc_lbaas_network.existing_lb.id
}

# Output load balancer details
output "lb_details" {
  value = {
    name       = data.mgc_lbaas_network.existing_lb.name
    visibility = data.mgc_lbaas_network.existing_lb.visibility
    backends   = data.mgc_lbaas_network_backends.all_backends.backends
    listeners  = data.mgc_lbaas_network_listeners.all_listeners.listeners
  }
}
```
