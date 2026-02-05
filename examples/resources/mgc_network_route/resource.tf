resource "mgc_network_route" "example" {
  vpc_id           = "your-vpc-id"
  port_id          = "your-port-id"
  cidr_destination = "xxx.xxx.xxx.xxx/xx"
  description      = "Route example"
}
