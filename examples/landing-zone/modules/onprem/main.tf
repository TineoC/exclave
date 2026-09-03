# On-premises landing zone.
#
# The honest case, and the one that proves whether the interface is real: there
# is no organisation, no folder tree and no billing account. If the interface
# only works when a hyperscaler supplies the hierarchy, it was never portable —
# it was GCP with the names filed off.
#
# Fill the body with Cluster API, RKE2 or your existing provisioning:
#   https://cluster-api.sigs.k8s.io/
#
# Interface and concept mapping only — see the note in gcp/main.tf.

terraform {
  required_version = ">= 1.5"
}

locals {
  # With no account hierarchy, the cluster itself is the isolation boundary.
  # Contracts do not share one; separation is physical or hypervisor-level.
  isolation_scope_id = "${var.name}-${var.classification}"

  # parent_scope is unused. topology is still meaningful — it records whether
  # the hardware is ours or the customer's, which drives who holds the console.
  parent = ""

  cluster_name     = "${var.name}-rke2"
  cluster_endpoint = "https://${local.cluster_name}.${var.region}.internal:6443"

  # No managed identity service. Keycloak — already a baseline component — is
  # the OIDC issuer, which is why the baseline ships it rather than assuming a
  # cloud provides one.
  oidc_issuer_url       = "https://keycloak.${var.name}.internal/realms/${var.name}"
  workload_identity_ref = "keycloak:${var.name}"

  cluster_ca_certificate = "" # from the cluster's generated CA
  kubeconfig_command     = "kubectl config use-context ${local.cluster_name}"

  # Harbor is not optional here. Air-gapped sites have no registry to fall back
  # to, so the one output that never varies is the one that matters most.
  registry_url = "harbor.${var.name}.internal"

  audit_sink_destination = var.audit_sink != "" ? var.audit_sink : "file:///var/log/kubernetes/audit.log"

  # Sizes are node labels rather than machine types; capacity is whatever the
  # rack has.
  node_size_map = { small = "size=small", medium = "size=medium", large = "size=large" }

  # Guardrails: Kyverno in enforce mode, since there is no cloud policy engine
  # above the cluster to inherit from. This is the case that shows why the
  # baseline carries its own policy engine instead of relying on the provider.
}
