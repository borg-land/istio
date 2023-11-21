{{- define "l7_config" }}
    enabled: {{ list nil true | has .Values.l7Telemetry.enabled }}
    sameNode: {{ list nil true | has .Values.l7Telemetry.sameNode }}
    metrics:
        enabled: {{ list nil true | has .Values.l7Telemetry.metrics.enabled }}
    accessLog:
        enabled: {{ list nil true | has .Values.l7Telemetry.accessLog.enabled }}
        file: {{ quote .Values.l7Telemetry.accessLog.file | default "" }}
        fileBoundSize: {{ .Values.l7Telemetry.accessLog.fileBoundSize | default 64000 }}
        requestHeaders: {{ .Values.l7Telemetry.accessLog.requestHeaders | default list }}
    distributedTracing:
        enabled: {{ list nil true | has .Values.l7Telemetry.accessLog.enabled }}
        otlpEndpoint: {{ quote .Values.l7Telemetry.distributedTracing.otlpEndpoint | default "http://opentelemetry-collector.istio-system:4317" }}

{{- end }}