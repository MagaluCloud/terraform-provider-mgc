# Network contracts

`fixture_test.go` implements VPC create/read/delete, retaining IDs and mutation
counts. It rejects unexpected API calls and duplicate creation while an object
is still present.

`lifecycle_test.go` covers create, repeated apply, CLI/HCL import and recovery
from a 403 without replacing the existing VPC. `regression_test.go` covers
external deletion and failure after a successful create response. Terraform
must retain a partial create so its next apply explicitly replaces the tainted
ID; it must not silently create an additional unmanaged VPC.

Shared polling deadline/cancellation tests live in `mgc/utils/poll_test.go`.
