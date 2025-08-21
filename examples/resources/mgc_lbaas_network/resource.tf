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
      action           = "ALLOW"
      ethertype        = "IPv4"
      protocol         = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]
}
