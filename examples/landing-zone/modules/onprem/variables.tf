# Landing zone interface — inputs.
#
# Identical across every provider. A contract descriptor fills these in and does
# not know or care which implementation consumes them. Provider-specific detail
# is confined to `parent_scope` and `billing_ref`, which are opaque strings whose
# meaning is documented per module.
#
# Do not add a provider-specific variable here. If one provider needs something
# the others do not, it belongs in that module's own optional variables, not in
# the shared interface — the moment this file differs between modules, the
# portability claim is dead.

variable "name" {
  description = "Contract name. Becomes the isolation scope name and the label applied to everything in it."
  type        = string
}

variable "classification" {
  description = "DoD impact level. IL1 and IL3 were consolidated by the Cloud Computing SRG and are not valid."
  type        = string
  validation {
    condition     = contains(["il2", "il4", "il5", "il6"], var.classification)
    error_message = "classification must be il2, il4, il5 or il6."
  }
}

variable "topology" {
  description = <<-EOT
    Who owns the hierarchy above this landing zone.

      contractor-owned  we bootstrapped the whole tree and own the root
      customer-owned    they gave us a scope inside their own org

    Everything below the parent scope is identical either way. This variable
    exists so the module can assert the right permissions, not so it can build
    two different things.
  EOT
  type        = string
  validation {
    condition     = contains(["contractor-owned", "customer-owned"], var.topology)
    error_message = "topology must be contractor-owned or customer-owned."
  }
}

variable "parent_scope" {
  description = "Where this landing zone attaches. GCP: folder ID. AWS: organizational unit ID. Azure: management group ID. On-prem: unused, pass an empty string."
  type        = string
}

variable "billing_ref" {
  description = "GCP: billing account ID. AWS: payer account ID. Azure: enrollment/billing scope. On-prem: cost centre, or empty."
  type        = string
  default     = ""
}

variable "region" {
  description = "Primary region. On-prem: site or datacentre identifier."
  type        = string
}

variable "network_cidr" {
  description = "CIDR for the contract's private network. Must not overlap any other contract."
  type        = string
}

variable "egress" {
  description = "Outbound posture: none for air-gapped sites, nat for proxied egress, restricted for an allow-listed set."
  type        = string
  default     = "restricted"
  validation {
    condition     = contains(["none", "nat", "restricted"], var.egress)
    error_message = "egress must be none, nat or restricted."
  }
}

variable "admin_principals" {
  description = "Identities granted administrative access to the isolation scope. Group identifiers, never individual users."
  type        = list(string)
  default     = []
}

variable "cluster_version" {
  description = "Kubernetes minor version, e.g. 1.29. The baseline declares the range it supports; this must fall inside it."
  type        = string
}

variable "cluster_node_pools" {
  description = "Normalised node pool spec. Providers map size to their own machine families."
  type = list(object({
    name   = string
    size   = string # small | medium | large — mapped per provider
    min    = number
    max    = number
    taints = optional(list(string), [])
    spot   = optional(bool, false)
  }))
}

variable "audit_sink" {
  description = "Destination for audit logs. Provider-specific URI; the module guarantees logs arrive there, not how."
  type        = string
  default     = ""
}

variable "labels" {
  description = "Labels or tags applied to every resource. Contract name and classification are added automatically."
  type        = map(string)
  default     = {}
}
