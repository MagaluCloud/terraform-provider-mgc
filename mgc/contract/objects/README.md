# Object Storage contracts

The fixture implements the S3 calls used by the provider: bucket location,
PUT/HEAD/DELETE object and retention lookup. It stores the actual uploaded
bytes and counts uploads/deletions independently of Terraform state.

SDK v1.19.0 accepts only fixed S3 hostnames. `s3Server` aliases the southeast
hostname to the local TLS server before DNS; it never connects to that host.
Authentication and multipart behavior are outside this fixture's scope.

`lifecycle_test.go` covers equal-length content and source-path updates,
no-op applies, missing-object recovery, CLI import and known retention state.
Editing bytes at the same `source` path without changing any configured value
still produces no Terraform diff. Use `content = file(...)` for tracked text;
these fixes do not introduce a source-hash feature.
