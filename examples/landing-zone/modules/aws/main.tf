# AWS landing zone.
#
# Fill the body from the Landing Zone Accelerator or Control Tower Account
# Factory for Terraform rather than hand-rolling Organizations plumbing:
#   https://aws.amazon.com/solutions/implementations/landing-zone-accelerator-on-aws/
#
# Interface and concept mapping only — see the note in gcp/main.tf.

terraform {
  required_version = ">= 1.5"
}

locals {
  # The unit of isolation is an account. This is the sharpest difference from
  # GCP: accounts are heavyweight, slow to create and hard to delete, so a
  # contract gets one and its environments become namespaces rather than peers.
  isolation_scope_id = "${var.name}-${var.classification}"

  # parent_scope is an organizational unit ID: "ou-a1b2-c3d4e5f6".
  # Guardrails attach here as Service Control Policies, which is the closest
  # analogue to GCP org policy and Azure Policy.
  parent = var.parent_scope

  cluster_name     = "${var.name}-eks"
  cluster_endpoint = "https://${local.cluster_name}.${var.region}.eks.internal"

  # EKS exposes an OIDC provider that IAM trusts; IRSA is the workload identity
  # mechanism the baseline binds against.
  oidc_issuer_url       = "https://oidc.eks.${var.region}.amazonaws.com/id/${local.cluster_name}"
  workload_identity_ref = "arn:aws:iam::ACCOUNT_ID:oidc-provider/oidc.eks.${var.region}.amazonaws.com/id/${local.cluster_name}"

  cluster_ca_certificate = "" # from aws_eks_cluster.certificate_authority
  kubeconfig_command     = "aws eks update-kubeconfig --name ${local.cluster_name} --region ${var.region}"

  registry_url = "harbor.${var.name}.internal"

  audit_sink_destination = var.audit_sink != "" ? var.audit_sink : "s3://${var.name}-audit-logs/AWSLogs/"

  node_size_map = { small = "m6i.xlarge", medium = "m6i.2xlarge", large = "m6i.4xlarge" }

  # Guardrails: SCPs denying root usage, region restrictions, and mandatory
  # CloudTrail. GuardDuty and Config are enabled at the OU, not per account.
}
