apiVersion: apps/v1
kind: Deployment
metadata:
  name: &name {{ include "ip-dns.name" . }}
  namespace: {{.Release.Namespace}}
  labels: &labels
    app: *name
spec:
  selector:
    matchLabels: *labels
  template:
    metadata:
      labels: *labels
    spec:
      hostNetwork: false
      securityContext:
        runAsUser: 0
      containers:
        - name: main
          image: {{ include "ip-dns.image" .}}
          imagePullPolicy: {{.Values.image.pullPolicy}}
          args:
            - --tcp-addr
            - :5953
            - --udp-addr
            - :5953
            - --http-addr
            - :8053
            - --domain-suffix
            - {{.Values.domainSuffix}}
          securityContext:
            privileged: false
