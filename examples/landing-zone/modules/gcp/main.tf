# GCP landing zone.
#
# Fill the body from Cloud Foundation Fabric — Google's own Terraform modules —
# rather than writing project and folder plumbing by hand:
#   https://github.com/GoogleCloudPlatform/cloud-foundation-fabric
#
# What ships here is the interface and the concept mapping. The resource blocks
# are deliberately absent: untested infrastructure code that looks real is worse
# than an interface that is honest about being one.

terraform {
  required_version = ">= 1.5"
}

locals {
  # The unit of isolation is a project. Billing attaches here, IAM inherits from
  # the folder above, and deleting it deletes everything in the contract.
  isolation_scope_id = "${var.name}-${var.classification}"

  # parent_scope is a folder ID: "folders/447120983001".
  #   contractor-owned  the folder sits under our organisation
  #   customer-owned    they created it in theirs and granted us roles on it
  # Nothing below this line differs between the two.
  parent = var.parent_scope

  cluster_name     = "${var.name}-gke"
  cluster_endpoint = "https://${local.cluster_name}.${var.region}.gke.internal"

  # GKE issues its own OIDC and Workload Identity pool; the baseline federates
  # Kubernetes service accounts against them so nothing needs a static key.
  oidc_issuer_url       = "https://container.googleapis.com/v1/projects/${local.isolation_scope_id}/locations/${var.region}/clusters/${local.cluster_name}"
  workload_identity_ref = "${local.isolation_scope_id}.svc.id.goog"

  cluster_ca_certificate = "" # from google_container_cluster.master_auth
  kubeconfig_command     = "gcloud container clusters get-credentials ${local.cluster_name} --region ${var.region} --project ${local.isolation_scope_id}"

  # Harbor runs in-cluster and is the registry everything pulls from. Artifact
  # Registry may back it, but consumers only ever see the Harbor endpoint.
  registry_url = "harbor.${var.name}.internal"

  audit_sink_destination = var.audit_sink != "" ? var.audit_sink : "logging.googleapis.com/projects/${local.isolation_scope_id}/logs/cloudaudit"

  # Machine families are the provider's business, not the caller's.
  node_size_map = { small = "e2-standard-4", medium = "n2-standard-8", large = "n2-standard-16" }

  # Guardrails belong here as org policy constraints: disable service account
  # key creation, require Shielded VMs, restrict external IPs, and turn on
  # Binary Authorization so unsigned images cannot run.
}
