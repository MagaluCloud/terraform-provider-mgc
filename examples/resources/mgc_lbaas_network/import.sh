# Load balancer in the default project
terraform import mgc_lbaas_network.http_proxy my_lbaas_id

# Load balancer in a project
terraform import mgc_lbaas_network.http_proxy project/550e8400-e29b-41d4-a716-446655440000/my_lbaas_id
