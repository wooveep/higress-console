{{/*
Expand the name of the chart.
*/}}
{{- define "aigateway-console.name" -}}
{{- default .Chart.Name .Values.name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "aigateway-console.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "aigateway-console.labels" -}}
helm.sh/chart: {{ include "aigateway-console.chart" . }}
{{ include "aigateway-console.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "aigateway-console.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aigateway-console.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "aigateway-console.controller.jwtPolicy" -}}
{{- if semverCompare ">=1.21-0" .Capabilities.KubeVersion.GitVersion }}
{{- .Values.global.jwtPolicy | default "third-party-jwt" }}
{{- else }}
{{- print "first-party-jwt" }}
{{- end }}
{{- end }}

{{- define "aigateway-console.clusterDomain" -}}
{{- .Values.global.proxy.clusterDomain | default "cluster.local" -}}
{{- end }}

{{- define "aigateway-console.serviceHost" -}}
{{- $ctx := .context -}}
{{- printf "%s.%s.svc.%s" .service $ctx.Release.Namespace (include "aigateway-console.clusterDomain" $ctx) -}}
{{- end }}

{{/*
Admin Password
*/}}
{{- define "aigateway-console.adminPassword" -}}
app.kubernetes.io/name: {{ include "aigateway-console.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create a default fully qualified app name for Grafana.
*/}}
{{- define "aigateway-console-grafana.name" -}}
{{- $consoleName := include "aigateway-console.name" . }}
{{- printf "%s-grafana" ($consoleName | trunc 55) }}
{{- end }}

{{- define "aigateway-console-grafana.path" -}}
/grafana
{{- end }}

{{/*
Create a default fully qualified app name for Prometheus.
*/}}
{{- define "aigateway-console-prometheus.name" -}}
{{- $consoleName := include "aigateway-console.name" . }}
{{- printf "%s-prometheus" ($consoleName | trunc 52) }}
{{- end }}

{{- define "aigateway-console-prometheus.path" -}}
/prometheus
{{- end }}

{{/*
Create a default fully qualified app name for cert-manager
*/}}
{{- define "aigateway-console-cert-manager.name" -}}
{{- $consoleName := include "aigateway-console.name" . }}
{{- printf "%s-cert-manager" ($consoleName | trunc 52) }}
{{- end }}

{{/*
Create a default fully qualified app name for promtail
*/}}
{{- define "aigateway-console-promtail.name" -}}
{{- $consoleName := include "aigateway-console.name" . }}
{{- printf "%s-promtail" ($consoleName | trunc 52) }}
{{- end }}

{{/*
Create a default fully qualified app name for loki
*/}}
{{- define "aigateway-console-loki.name" -}}
{{- $consoleName := include "aigateway-console.name" . }}
{{- printf "%s-loki" ($consoleName | trunc 52) }}
{{- end }}

{{/*
Create the default plugin server URL pattern.
*/}}
{{- define "aigateway-console.pluginServer.urlPattern" -}}
{{- if .Values.pluginServer.urlPattern -}}
{{- .Values.pluginServer.urlPattern -}}
{{- else -}}
{{- printf "http://%s/plugins/${name}/${version}/plugin.wasm" (include "aigateway-console.serviceHost" (dict "context" . "service" (.Values.pluginServer.serviceName | default "aigateway-plugin-server"))) -}}
{{- end -}}
{{- end }}
