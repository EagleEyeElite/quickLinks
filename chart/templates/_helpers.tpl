{{/*
Fixed resource names — this chart deploys exactly one quick-links stack, and the
Service names + Traefik middleware references depend on these names, so they are
NOT release-name-derived. (Matches the hanna chart's convention.)
*/}}

{{- define "quicklinks.labels" -}}
app.kubernetes.io/part-of: quick-links
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}


{{/*
Password of the read-only grafana_ro role. Read at deploy time from the Secret
that personal-infrastructure (Pulumi) owns, so the value never sits in this repo
or in values. `lookup` needs a live cluster: `helm template` / `helm lint` without
one get a placeholder instead of failing; a real install/upgrade fails loudly if
the Secret is missing.
*/}}
{{- define "quicklinks.grafanaRoPassword" -}}
{{- $ref := .Values.grafana.roPasswordSecret -}}
{{- $s := lookup "v1" "Secret" .Release.Namespace $ref.name -}}
{{- if $s -}}
{{- index $s.data $ref.key | required (printf "Secret %s has no key %q" $ref.name $ref.key) | b64dec -}}
{{- else if .Values.grafana.offlineRender -}}
OFFLINE-RENDER-PLACEHOLDER
{{- else -}}
{{- fail (printf "Secret %s/%s not found. It is created by personal-infrastructure (pulumi up, src/self-hosted/gateway.ts). For offline rendering pass --set grafana.offlineRender=true" .Release.Namespace $ref.name) -}}
{{- end -}}
{{- end -}}
