# Kubernetes manifests

`deployment.yaml` contains `${VERSION}`, `${BUILD_ID}` and `${GIT_SHA}`
placeholders. They are **not** expanded by Kubernetes — render before applying:

```sh
export VERSION=1.2.3 BUILD_ID=ci-$(date +%s) GIT_SHA=$(git rev-parse HEAD)
kubectl kustomize k8s/ | kubectl apply -f -
# or, without kustomize:
envsubst < k8s/deployment.yaml | kubectl apply -f -
```

Before first deploy, edit the `crm-secrets` Secret: set a real
`CRM_OWNER_EMAIL`, `CRM_OWNER_PASSWORD` and `CRM_ENCRYPTION_KEY`
(`openssl rand -hex 32`). The Ingress host `crm.yourdomain.com` and the
cert-manager issuer must match your cluster.

The pod is annotated for Prometheus scraping (`prometheus.io/path: /metrics`,
port 8080); the Go server exposes `/metrics` directly.
