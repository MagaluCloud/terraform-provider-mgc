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

Acceptance tests (`TestAcc*`) exercise the full Terraform lifecycle (plan, apply,
import, in-place update, refresh and destroy) through `terraform-plugin-testing`.
They are endpoint-agnostic: the API they target is decided solely by
`MGC_ENDPOINT`, which may point to a fake API that simulates Magalu Cloud or to
the production API. The tests behave identically against either.

```bash
# Against a locally running fake API (fast; any valid UUIDv4 works as key)
MGC_ENDPOINT=http://localhost:8080 \
MGC_API_KEY=8a1f47b0-3a4e-4b91-9e55-1c2d3e4f5a6b \
MGC_POLLING_INTERVAL=200ms \
make testacc

# Against production (creates real, billable resources)
MGC_ENDPOINT=<production API root URL> MGC_API_KEY=<real key> make testacc
```

Optional variables: `MGC_REGION` (default `br-se1`), `MGC_K8S_VERSION` /
`MGC_K8S_VERSION_UPGRADE` (cluster versions used by the Kubernetes lifecycle
test; when unset, `version` is omitted from the config), and
`MGC_POLLING_INTERVAL` (Go duration overriding the 10s wait between status
checks — useful against a fake API that transitions states instantly).

Without `TF_ACC` set (exported automatically by `make testacc`), `TestAcc*`
tests are skipped, so `make go-test` and CI remain unaffected.

##### Recording and replaying with VCR

Acceptance tests can record their HTTP traffic into a **cassette** once and then
**replay** it on every later run, so regression checks need no infrastructure,
no credentials and finish in milliseconds. This is driven by `MGC_VCR_MODE`:

| `MGC_VCR_MODE` | Behaviour |
| --- | --- |
| unset / `auto` | Record once: replay if a cassette exists, otherwise hit the API and record it. This is what you want for a brand-new test. |
| `record` | Always hit the live API and (re)record. Requires `MGC_ENDPOINT` + `MGC_API_KEY`. |
| `replay` | Replay only, from the committed cassette. Fails if the cassette is missing. No network. |
| `off` / `live` | Bypass VCR entirely and talk to the real API. |

```bash
# Record cassettes against a live/fake API (run once per new or changed test)
MGC_ENDPOINT=http://localhost:8080 MGC_API_KEY=<uuid> MGC_K8S_VERSION=<v> \
MGC_POLLING_INTERVAL=200ms make testacc-record

# Replay from committed cassettes — no infra, no secrets
make testacc-replay
```

Cassettes are written under each service package's `testdata/cassettes/`
directory (e.g. `mgc/kubernetes/testdata/cassettes/<TestName>.yaml`) and are
meant to be committed. Credentials (`Authorization`, `X-Api-Key`) are scrubbed
before the cassette is written, and random resource names are seeded from the
test name so record and replay stay byte-for-byte identical.

> **Replay needs the same non-secret inputs used at record time.** Values that
> end up in request bodies — `MGC_K8S_VERSION`, `MGC_K8S_VERSION_UPGRADE`,
> `MGC_REGION` — must match the recording, because the request matcher compares
> request bodies. `make testacc-replay` defaults `MGC_ENDPOINT`/`MGC_API_KEY`/
> `MGC_POLLING_INTERVAL` to harmless placeholders but passes these through from
> your environment.

To wire a new service's acceptance test into VCR, build the recorder with
`acctest.NewVCR(t)` and use `vcr.ProtoV6ProviderFactories()`, `vcr.RandomName()`
and `vcr.HTTPClient()` (for any SDK client a check builds directly, e.g.
`CheckDestroy`) — see `mgc/kubernetes/resource_cluster_acc_test.go`.

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
