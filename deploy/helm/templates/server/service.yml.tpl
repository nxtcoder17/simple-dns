apiVersion: v1
kind: Service
metadata:
  name: {{include "ip-dns.name" . }}
  namespace: {{.Release.Namespace}}
spec:
  selector:
    app: {{include "ip-dns.name" .}}
  ports:
    - name: tcp-53
      protocol: TCP
      port: 53
      targetPort: 5953
    - name: udp-53
      protocol: UDP
      port: 53
      targetPort: 5953
    - name: dns-http
      protocol: TCP
      port: 80
      targetPort: 8053
  type: ClusterIP
