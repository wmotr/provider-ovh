# wmotr provider-ovh fork

This fork tracks Edixos v2.19.1 (commit c899223b00b3f2a835a1a5503a96b6cf32d86b63).
It changes only the SDK external-name mappings for `ovh_cloud_project_kube` and
`ovh_cloud_project_kube_nodepool`: SDK Read receives the raw provider UUID,
not the composite identifier used by Terraform CLI ImportState. The name
initializer remains disabled. No generated API or CRD schema changes are intended.

`Provider CI` runs on `gh-runners-k8s`, regenerates sources, lints, tests and builds
packages for linux/amd64 and linux/arm64. Internal pull requests validate without
publication. After a reviewed PR merges, main publishes the tested packages to
`ghcr.io/wmotr/provider-ovh:v2.19.1-wmotr.1.g<commit-prefix>`. Deployment references
belong in a separate `wm-hosting-configs` PR and should pin the published digest.

The runtime is embedded in each Crossplane package by the upstream build system.
Only the package publication target runs: a separate runtime push must not
replace the package tag. Registry pulls can use the existing Vault-managed GHCR
credential; no OVH API credential participates in building or publishing.

Upstream workflows are preserved under `.github/upstream-workflows/` and do not
run in this fork. In particular, CI does not deploy OVH examples or publish to the
Edixos marketplace. The upstream license and authorship remain intact.
