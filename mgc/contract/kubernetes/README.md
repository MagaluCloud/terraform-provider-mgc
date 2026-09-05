# Kubernetes contracts

The fixture stores cluster version, description and inherited network values.
It counts POST/PATCH/DELETE calls so a successful-looking state update cannot
hide replacement or a missing API update.

Lifecycle scenarios cover in-place upgrades, description changes, stable subnet
sets, inherited values, CLI/HCL import and prevent_destroy. The deletion case
accepts DELETE but fails the following GET: Terraform must retain the resource
and retry deletion after recovery, not discard its state.

Polling unit tests separately cover missing status, expected target version,
deadlines and a transient 404 after creation. These fixtures do not model actual
cluster provisioning, workload health or API authentication.

The upgrade/description fixture returns the old running cluster on the first
GET after each PATCH. Update must wait for version, description and allowed
CIDRs to converge; status alone is insufficient. Production one-minute polling
intervals are used without test-only acceleration. Unit tests additionally
separate stale description from stale CIDRs and check list ordering.
