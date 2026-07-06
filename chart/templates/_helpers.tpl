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
