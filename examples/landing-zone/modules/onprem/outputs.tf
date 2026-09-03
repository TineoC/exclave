# Landing zone interface — outputs.
#
# This is the contract that makes everything above the landing zone
# cloud-agnostic. The baseline, the GitOps controller and exclave consume these
# outputs; none of them ever learns which cloud produced them.
#
# A provider that cannot satisfy an output has not implemented the interface.
# Adding an output here means implementing it in all four modules.

output "isolation_scope_id" {
  description = "The unit of isolation. GCP project, AWS account, Azure subscription, or on-prem cluster identifier. Billing and IAM boundaries follow this."
  value       = local.isolation_scope_id
}

output "cluster_name" {
  description = "Kubernetes cluster name."
  value       = local.cluster_name
}

output "cluster_endpoint" {
  description = "Kubernetes API server endpoint. Private in every implementation."
  value       = local.cluster_endpoint
}

output "cluster_ca_certificate" {
  description = "Cluster CA, base64-encoded."
  value       = local.cluster_ca_certificate
  sensitive   = true
}

output "kubeconfig_command" {
  description = "The command an operator runs to obtain credentials. Provider-specific by nature, which is why it is a string rather than a mechanism."
  value       = local.kubeconfig_command
}

output "oidc_issuer_url" {
  description = "Cluster OIDC issuer, for workload identity federation. The baseline binds service accounts against this."
  value       = local.oidc_issuer_url
}

output "workload_identity_ref" {
  description = "Provider handle for workload identity: GCP workload identity pool, AWS IAM OIDC provider ARN, Azure federated credential scope. Opaque to consumers."
  value       = local.workload_identity_ref
}

output "registry_url" {
  description = "Harbor endpoint serving this contract. Harbor runs identically on every provider, so this is the one output whose shape never varies."
  value       = local.registry_url
}

output "audit_sink_destination" {
  description = "Where audit logs actually land. An assessor will ask; this is the answer."
  value       = local.audit_sink_destination
}

output "classification" {
  description = "Echoed back so downstream modules and the exclave contract descriptor cannot disagree with the landing zone about what this site is."
  value       = var.classification
}
