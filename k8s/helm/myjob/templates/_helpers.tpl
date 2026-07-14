{{/*
Common labels for all resources
*/}}
{{- define "myjob.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{/*
Selector labels (used in matchLabels)
*/}}
{{- define "myjob.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
API image name
*/}}
{{- define "myjob.api.image" -}}
{{- if .Values.global.imageRegistry -}}
{{- .Values.global.imageRegistry }}/{{ .Values.api.image.repository }}:{{ .Values.api.image.tag }}
{{- else -}}
{{- .Values.api.image.repository }}:{{ .Values.api.image.tag }}
{{- end -}}
{{- end }}

{{/*
Worker image name
*/}}
{{- define "myjob.worker.image" -}}
{{- if .Values.global.imageRegistry -}}
{{- .Values.global.imageRegistry }}/{{ .Values.worker.image.repository }}:{{ .Values.worker.image.tag }}
{{- else -}}
{{- .Values.worker.image.repository }}:{{ .Values.worker.image.tag }}
{{- end -}}
{{- end }}

{{/*
Browser Agent image name
*/}}
{{- define "myjob.browserAgent.image" -}}
{{- if .Values.global.imageRegistry -}}
{{- .Values.global.imageRegistry }}/{{ .Values.browserAgent.image.repository }}:{{ .Values.browserAgent.image.tag }}
{{- else -}}
{{- .Values.browserAgent.image.repository }}:{{ .Values.browserAgent.image.tag }}
{{- end -}}
{{- end }}

{{/*
Frontend image name
*/}}
{{- define "myjob.frontend.image" -}}
{{- if .Values.global.imageRegistry -}}
{{- .Values.global.imageRegistry }}/{{ .Values.frontend.image.repository }}:{{ .Values.frontend.image.tag }}
{{- else -}}
{{- .Values.frontend.image.repository }}:{{ .Values.frontend.image.tag }}
{{- end -}}
{{- end }}

{{/*
PostgreSQL host (internal service)
*/}}
{{- define "myjob.postgres.host" -}}
{{ .Release.Name }}-postgres
{{- end }}

{{/*
Redis host (internal service)
*/}}
{{- define "myjob.redis.host" -}}
{{ .Release.Name }}-redis
{{- end }}
