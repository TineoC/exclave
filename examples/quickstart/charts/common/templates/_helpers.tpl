{{/*
Registry indirection. Every image reference in the product resolves through one
prefix so a customer can mirror into Harbor, Nexus, Artifactory or an internal
GitLab registry without patching charts.

Digest is preferred over tag: tags are mutable, and an assessor will ask. The tag
path exists for local development against images loaded straight into a kind
cluster, where no registry digest exists yet.
*/}}
{{- define "common.image" -}}
{{- $reg := .Values.global.registry | default "" -}}
{{- $repo := .Values.image.repository -}}
{{- $ref := "" -}}
{{- if .Values.image.digest -}}
{{- $ref = printf "%s@%s" $repo .Values.image.digest -}}
{{- else if .Values.image.tag -}}
{{- $ref = printf "%s:%s" $repo .Values.image.tag -}}
{{- else -}}
{{- fail (printf "%s: set either image.digest (preferred) or image.tag" $repo) -}}
{{- end -}}
{{- if $reg -}}
{{- printf "%s/%s" (trimSuffix "/" $reg) $ref -}}
{{- else -}}
{{- $ref -}}
{{- end -}}
{{- end -}}

{{- define "common.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "common.labels" -}}
app.kubernetes.io/name: {{ include "common.name" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | default .Chart.Version | quote }}
app.kubernetes.io/part-of: {{ .Values.global.product | default "acme-platform" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "common.selectorLabels" -}}
app.kubernetes.io/name: {{ include "common.name" . }}
{{- end -}}
