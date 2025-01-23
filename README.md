## Magalu Cloud Provider

The MGC provider allows you to use Terraform to manage your resources on Magalu Cloud.

With the provider, you can manage:

- VPCs (subnets, security rules, public IPs)
- Virtual Machines (instances, snapshots)
- Kubernetes (clusters, node pools)
- DBaaS (instances, replications)
- Block Storage (volumes, snapshots, VM attachment)
- Object Storage

The provider is currently under development, so new Magalu Cloud resources will be supported soon.

# References
[Magalu Cloud Official Documentation](https://docs.magalu.cloud/docs/terraform/overview)

[Magalu Cloud Terraform Provider](https://registry.terraform.io/providers/MagaluCloud/mgc/latest)

# Participate
- You can contribute to an [open issue](https://github.com/MagaluCloud/terraform-provider-mgc/issues) to report a bug or suggest improvements and new features
- You can open tópic in our [Dicussions Forum](https://github.com/MagaluCloud/terraform-provider-mgc/discussions)
- See our roadmap in [projects](https://github.com/orgs/MagaluCloud/projects/2/views/7)

## Contributing

### pre-commit

We use [pre-commit](https://pre-commit.com/) to install git hooks and enforce
lint, formatting, tests, commit messages and others. This tool depends on
Python as well. On pre-commit we enforce:

* On `commit-msg` for all commits:
  * [Conventional commit](https://www.conventionalcommits.org/en/v1.0.0/) pattern
    with [commitzen](https://github.com/commitizen/cz-cli)
* On `pre-commit` for Go files:
  * Complete set of [golangci-lint](https://golangci-lint.run/): `errcheck`,
    `gosimple`, `govet`, `ineffasign`, `staticcheck`, `unused`
* On `pre-commit` for Python files:
  * `flake8` and `black` enforcing pep code styles

#### Installation

#### Mac

```sh
brew install pre-commit
```

#### pip

```sh
pip install pre-commit
```


For other types of installation, check their
[official doc](https://pre-commit.com/#install).

#### Configuration

After installing, the developer must configure the git hooks inside its clone:

```sh
pre-commit install
```

### Linters

We install the go linters via `pre-commit`, so it is automatically run by the
pre-commit git hook. However, if one wants to run standalone it can be done via:

```sh
pre-commit run golangci-lint
```

### Run all

Run pre-commit without any file modified:

```sh
pre-commit run -a
```