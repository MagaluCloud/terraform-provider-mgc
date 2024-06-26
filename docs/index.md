---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc Provider"
subcategory: ""
description: |-
  Terraform Provider for Magalu Cloud
---

# Terraform Provider for Magalu Cloud

Magalu Cloud (MGC) it's the first **Brazilian** Cloud with global scale. We speak your language, we understand your reality and we are ready to accelerate your business.

The Magalu Cloud provider is used to configure your Magalu Cloud infrastructure.

With the provider you can manage:

- VPCs (subnets, security rules, public IPs)
- Virtual Machines (instances, snapshots)
- Kubernetes (clusters, nodepools)
- DBaaS (instances, replications)
- Block Storage (volumes, snapshots, VM attach)
- Object Storage (Buckets)
- The provider is in the development phase, so new Magalu Cloud features will be supported soon.

If you don't already have an account on Magalu Cloud (MGC) you can create one through our [console](https://console.magalu.cloud/login).


You can check our examples of using MGC Terraform in this [repository](https://github.com/MagaluCloud/terraform-examples/tree/main).

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `api_key` (String) Magalu API Key for authentication
- `object_storage` (Attributes) Specific Object Storage configuration (see [below for nested schema](#nestedatt--object_storage))
- `region` (String) Region

<a id="nestedatt--object_storage"></a>
### Nested Schema for `object_storage`

Optional:

- `key_pair` (Attributes) Specific Bucket Key Pair configuration (see [below for nested schema](#nestedatt--object_storage--key_pair))

<a id="nestedatt--object_storage--key_pair"></a>
### Nested Schema for `object_storage.key_pair`

Required:

- `key_id` (String) API Key ID
- `key_secret` (String) API Key Secret
<!-- schema generated by tfplugindocs -->
