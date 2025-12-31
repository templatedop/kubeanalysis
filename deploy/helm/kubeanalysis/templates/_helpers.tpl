{{/*
Expand the name of the chart.
*/}}
{{- define "kubeanalysis.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubeanalysis.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kubeanalysis.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kubeanalysis.labels" -}}
helm.sh/chart: {{ include "kubeanalysis.chart" . }}
{{ include "kubeanalysis.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kubeanalysis.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubeanalysis.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kubeanalysis.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kubeanalysis.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the PostgreSQL connection URL
*/}}
{{- define "kubeanalysis.postgresURL" -}}
{{- if .Values.vectorStore.postgres.url }}
{{- .Values.vectorStore.postgres.url }}
{{- else if .Values.postgresql.enabled }}
{{- printf "postgres://%s:%s@%s-postgresql:5432/%s?sslmode=disable" .Values.postgresql.auth.username .Values.postgresql.auth.password (include "kubeanalysis.fullname" .) .Values.postgresql.auth.database }}
{{- end }}
{{- end }}

{{/*
Create the Temporal host URL
*/}}
{{- define "kubeanalysis.temporalHost" -}}
{{- if .Values.temporal.enabled }}
{{- printf "%s-temporal-frontend:7233" .Release.Name }}
{{- else }}
{{- .Values.config.temporal.host }}
{{- end }}
{{- end }}
