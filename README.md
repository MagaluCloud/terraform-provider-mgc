# Magalu Cloud Terraform Provider

[![Go Report Card](https://goreportcard.com/badge/github.com/MagaluCloud/terraform-provider-mgc)](https://goreportcard.com/report/github.com/MagaluCloud/terraform-provider-mgc)
[![HashiCorp Partner](https://img.shields.io/badge/HashiCorp-Technology%20Partner-7B42BC)](https://registry.terraform.io/providers/MagaluCloud/mgc/latest)
[![Terraform Registry Downloads](https://img.shields.io/badge/dynamic/json?color=blue&label=downloads&query=%24.data.attributes.downloads&url=https%3A%2F%2Fregistry.terraform.io%2Fv2%2Fproviders%2Fmagalucloud%2Fmgc)](https://registry.terraform.io/providers/MagaluCloud/mgc/latest)
[![GitHub release (latest)](https://img.shields.io/github/v/release/MagaluCloud/terraform-provider-mgc)](https://github.com/MagaluCloud/terraform-provider-mgc/releases)
[![Go Checks](https://github.com/MagaluCloud/terraform-provider-mgc/actions/workflows/go-checks.yml/badge.svg?branch=main)](https://github.com/MagaluCloud/terraform-provider-mgc/actions/workflows/go-checks.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MagaluCloud/terraform-provider-mgc)](https://github.com/MagaluCloud/terraform-provider-mgc)
[![License](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

The official Terraform provider for Magalu Cloud, allowing you to manage your cloud infrastructure as code. As an official HashiCorp Partner, this provider follows Terraform best practices for reliability and user experience.

## Provider Features

The MGC provider gives you comprehensive control over your Magalu Cloud resources, including:

### Networking

- **Virtual Private Clouds (VPCs)** - Create isolated network environments
- **Subnets** - Segment your VPC networks
- **Security Rules** - Control traffic with fine-grained permissions
- **Public IPs** - Expose services to the internet

### Compute

- **Virtual Machines** - Deploy and manage instances with various sizes and configurations
- **VM Snapshots** - Create point-in-time backups of your instances
- **SSH Keys** - Securely access your virtual machines

### Kubernetes

- **Managed Kubernetes Clusters** - Deploy production-ready Kubernetes
- **Node Pools** - Scale your Kubernetes worker nodes

### Database as a Service (DBaaS)

- **Database Instances** - Deploy managed database services
- **Replication** - Configure high availability and read replicas

### Storage

- **Block Storage** - Persistent volumes for your instances
- **Volume Snapshots** - Point-in-time backups
- **VM Volume Attachment** - Connect storage to your virtual machines
- **Object Storage** - S3-compatible storage for unstructured data

## Usage Example

```hcl
terraform {
  required_providers {
    mgc = {
      source = "magalucloud/mgc"
    }
  }
}

provider "mgc" {
  region  = "br-ne1"
  api_key = var.api_key
}

variable "api_key" {
  type      = string
  sensitive = true
}

# Create a virtual machine instance
resource "mgc_virtual_machine_instances" "example_vm" {
  name              = "example-vm"
  machine_type      = "BV1-1-40"
  image             = "cloud-ubuntu-24.04 LTS"
  ssh_key_name      = "my-ssh-key"
}
```

## Releases and Versioning

The provider follows [semantic versioning](https://semver.org/) practices. You can find all releases, including pre-releases, release notes, and binaries at our [GitHub Releases page](https://github.com/MagaluCloud/terraform-provider-mgc/releases).

When specifying the provider version in your Terraform configurations, we recommend using version constraints to ensure compatibility:

```hcl
terraform {
  required_providers {
    mgc = {
      source  = "magalucloud/mgc"
      version = "~> 0.32.0"
    }
  }
}
```

## Documentation

For complete usage documentation and examples, visit:

- [Magalu Cloud Official Documentation](https://docs.magalu.cloud/docs/terraform/overview)
- [Terraform Registry Documentation](https://registry.terraform.io/providers/MagaluCloud/mgc/latest/docs)

## Local Development

### Building the Provider

1. Clone the repository
2. Install dependencies

- Install [Go](https://golang.org/dl/) (version 1.26.3 or higher)
- Install [GoReleaser](https://goreleaser.com/)
- Install [Terraform](https://www.terraform.io/downloads)
- Install [Make](https://www.gnu.org/software/make/)
- Install [Git](https://git-scm.com/downloads)

3. Run `make build` to build the provider locally

```bash
# Clone the repo
git clone https://github.com/MagaluCloud/terraform-provider-mgc.git
cd terraform-provider-mgc

# Build the provider
make build
```

### Testing

Before submitting contributions, please run:

```bash
# Run pre-commit checks
make before-commit

# Run all tests
make go-test
```

#### Acceptance tests

The `TestAcc*` tests cover the entire Terraform lifecycle—plan, apply, import,
update, refresh, and destroy—using the `terraform-plugin-testing`. These tests require that the
`terraform` binary be in the `PATH` and are skipped unless the `TF_ACC` variable is set,
which is the purpose of the `testacc-*` tests.

A normal run replays previously recorded HTTP traffic from a **replay**.
No infrastructure, no credentials—all in a matter of seconds

##### Run the suite

```bash
make cassettes-download            # fetch the published cassettes
make testacc-replay                # replay everything
make testacc-replay RUN=TestAccKubernetesCluster_basic
```

The cassettes are stored outside the repository, in `MGC_VCR_PATH` (the developer’s cache directory), in two layers:

- `main/` — the published set, the base that each playback reads from. Only CI should write to this when a PR is merged.
- `<branch>/` — only what your branch has rewritten. It is written using `make cassettes-publish`.

A replay first searches the branch layer and, if it doesn’t find what it’s looking for, falls back to `main/`.

##### Changing a test

Replay only works for identical requests; making changes requires re-recording the cassettes.

```bash
PROFILE=prod make testacc-record RUN=TestAccKubernetesCluster_basic
make testacc-replay                # condition: the entire test suite must pass
make cassettes-publish             # loads your branch layer
```

Four rules enforced by the tool:

- `RUN=` is required during recording. Rewriting everything is never implicit.
- Recording to a loopback endpoint is disallowed: a published cassette must
  come from the actual API; otherwise, playback no longer makes sense.
- `cassettes-publish` rejects `main`. The published set is only recorded by
  promoting a merged branch.
- A recording whose test failed is discarded, so a failed run never overwrites a valid cassette.

##### Iterating Against a Mock API

While writing a test, run it in real time against a local mock API. Nothing is recorded:

```bash
PROFILE=dev make testacc-live RUN=TestAccKubernetesCluster_basic
```

##### Add a test to a new service

Create the recorder with `acctest.NewVCR(t)`, and then use
`vcr.ProtoV6ProviderFactories()`, `vcr.RandomName()`, and `vcr.SDKClient()`.
See `mgc/kubernetes/resource_cluster_acc_test.go`.

## Contributing

We welcome contributions to the Magalu Cloud Terraform Provider!

1. **Report Issues**: Found a bug or have a feature request? [Open an issue](https://github.com/MagaluCloud/terraform-provider-mgc/issues)

2. **Submit PRs**: Contributions via pull requests are welcome. Please:

   - Fork the repository
   - Create a feature branch
   - Make your changes
   - Run `make before-commit` to verify
   - Submit a PR

3. **Join Discussions**: Participate in our [Discussions Forum](https://github.com/MagaluCloud/terraform-provider-mgc/discussions)

## License

This provider is released under the [Mozilla Public License 2.0](LICENSE).
