# Azure landing zone.
#
# Fill the body from the Cloud Adoption Framework enterprise-scale Terraform
# modules rather than assembling management groups by hand:
#   https://github.com/Azure/terraform-azurerm-caf-enterprise-scale
#
# Interface and concept mapping only — see the note in gcp/main.tf.

terraform {
  required_version = ">= 1.5"
}

locals {
  # The unit of isolation is a subscription. Resource groups sit inside it and
  # are closer to GCP projects in weight, but the billing and policy boundary is
  # the subscription, so that is what the interface returns.
  isolation_scope_id = "${var.name}-${var.classification}"

  # parent_scope is a management group ID. Azure Policy assigns here and
  # inherits downward, the same shape as SCPs and GCP org policy.
  parent = var.parent_scope

  cluster_name     = "${var.name}-aks"
  cluster_endpoint = "https://${local.cluster_name}.${var.region}.aks.internal"

  # AKS workload identity federates Kubernetes service accounts to Entra ID
  # applications through the cluster's OIDC issuer.
  oidc_issuer_url       = "https://${var.region}.oic.prod-aks.azure.com/${local.cluster_name}/"
  workload_identity_ref = "federated-credential:${local.isolation_scope_id}"

  cluster_ca_certificate = "" # from azurerm_kubernetes_cluster.kube_config
  kubeconfig_command     = "az aks get-credentials --name ${local.cluster_name} --resource-group ${var.name}-rg"

  registry_url = "harbor.${var.name}.internal"

  audit_sink_destination = var.audit_sink != "" ? var.audit_sink : "loganalytics://${var.name}-law"

  node_size_map = { small = "Standard_D4s_v5", medium = "Standard_D8s_v5", large = "Standard_D16s_v5" }

  # Guardrails: Azure Policy initiatives at the management group, Defender for
  # Cloud, and private endpoints for every PaaS dependency.
}
