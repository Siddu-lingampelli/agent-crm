# Monitoring stack

Optional `monitoring` compose profile (`docker compose --profile monitoring up`):

- Prometheus scrapes the CRM pod annotations (`prometheus.io/path: /metrics`, port 8080)
  in Kubernetes, or `crm:8080` directly via `prometheus.yml`'s `crm` job. Locally, add
  a `static_configs` target of `host.docker.internal:8080` if scraping the host build.
- Grafana is provisioned with the Prometheus and Loki datasources in
  `grafana/datasources/` and the dashboard in `grafana/dashboards/crm/`.
- Loki + Promtail aggregate container logs from the `crm-logs` volume.
- Alert rules live in `alerts.yml`; mount it under `/etc/prometheus/rules/` (the path
  referenced by `rule_files` in `prometheus.yml`).

## TLS certs

The `nginx` service mounts `./certs:/etc/nginx/certs:ro`. Provide
`certs/fullchain.pem` and `certs/privkey.pem` (e.g. from certbot/Let's Encrypt)
before starting the proxy, or remove the nginx service.
