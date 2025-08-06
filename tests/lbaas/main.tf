resource "mgc_loadbalancers_network" "lbmaroto" {
  description     = "Load Balancer para aplicação web"
  subnetpool_id   = "subnet-pool-12345"
  type            = "network"
  visibility      = "public"
  vpc_id          = "vpc-67890"

  # ACLs opcionais
  acls = [
    {
      action         = "allow"
      ethertype      = "IPv4"
      name           = "allow-http"
      protocol       = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    },
    {
      action         = "allow"
      ethertype      = "IPv4"
      name           = "allow-https"
      protocol       = "tcp"
      remote_ip_prefix = "0.0.0.0/0"
    }
  ]

  # Backends obrigatórios
  backends = [
    {
      balance_algorithm = "round_robin"
      description       = "Backend para servidores web"
      health_check_name = "web-health-check"
      name             = "web-backend"
      targets_type     = "ip"
      targets = [
        {
          ip_address = "192.168.1.10"
          port       = 80
        },
        {
          ip_address = "192.168.1.11"
          port       = 80
        }
      ]
    }
  ]

  # Health Checks opcionais
  health_checks = [
    {
      description                 = "Health check para servidores web"
      healthy_status_code         = 200
      healthy_threshold_count     = 2
      initial_delay_seconds       = 30
      interval_seconds           = 10
      name                       = "web-health-check"
      path                       = "/health"
      port                       = 80
      protocol                   = "HTTP"
      timeout_seconds            = 5
      unhealthy_threshold_count  = 3
    }
  ]

  # Listeners obrigatórios
  listeners = [
    {
      backend_name = "web-backend"
      description  = "Listener para HTTP"
      name         = "http-listener"
      port         = 80
      protocol     = "HTTP"
    },
    {
      backend_name        = "web-backend"
      description         = "Listener para HTTPS"
      name                = "https-listener"
      port                = 443
      protocol            = "HTTPS"
      tls_certificate_name = "web-certificate"
    }
  ]

  # TLS Certificates opcionais
  tls_certificates = [
    {
      certificate = "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKoK...\n-----END CERTIFICATE-----"
      description = "Certificado SSL para aplicação web"
      name        = "web-certificate"
      private_key = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSj...\n-----END PRIVATE KEY-----"
    }
  ]
}
