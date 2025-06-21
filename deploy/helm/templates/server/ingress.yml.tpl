apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    cert-manager.io/cluster-issuer: cluster-issuer
    nginx.ingress.kubernetes.io/proxy-body-size: 3000m
  name: {{include "ip-dns.name" .}}
  namespace: {{.Release.Namespace}}
spec:
  ingressClassName: nginx
  rules:
  - host: {{ include  "ip-dns.ingress.host" .}}
    http:
      paths:
      - backend:
          service:
            name: {{include "ip-dns.name" .}}
            port:
              number: 80
        path: /
        pathType: Prefix
  tls:
  - hosts:
    - {{include "ip-dns.ingress.host" .}}
    secretName: {{include "ip-dns.ingress.host" .}}-tls
