{{- define "ip-dns.name" }} ip-dns {{- end }}
{{- define "ip-dns.image" }} {{.Values.image.repository}}:{{.Values.image.tag}} {{- end }}
{{- define "ip-dns.ingress.host" }} {{.Values.ingress.host}} {{- end }}
