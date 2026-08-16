# MyJob Kubernetes Deployment

## Prerequisites

- k3s installed: `curl -sfL https://get.k3s.io | sh -`
- kubectl configured: `export KUBECONFIG=/etc/rancher/k3s/k3s.yaml`
- Helm 3.x installed

## Quick Start

1. Create secrets:
   ```bash
   kubectl create namespace myjob
   kubectl create secret generic myjob-database -n myjob --from-literal=POSTGRES_PASSWORD=yourpassword
   kubectl create secret generic myjob-auth -n myjob --from-literal=AUTH_JWT_SECRET=yourjwtsecret --from-literal=AUTH_PASSWORD_HASH=yourhash --from-literal=SESSION_SECRET=yoursessionsecret
   kubectl create secret generic myjob-redis -n myjob --from-literal=REDIS_PASSWORD=yourredispassword
   kubectl create secret generic myjob-livekit -n myjob --from-literal=LIVEKIT_API_KEY=yourkey --from-literal=LIVEKIT_API_SECRET=yoursecret
   kubectl create secret generic myjob-llm -n myjob --from-literal=OPENAI_API_KEY= --from-literal=ANTHROPIC_API_KEY=
   kubectl create secret generic myjob-microsoft -n myjob --from-literal=MS_TENANT_ID= --from-literal=MS_CLIENT_ID= --from-literal=MS_CLIENT_SECRET=
   ```

2. Deploy:
   ```bash
   helm install myjob ./k8s/helm/myjob --namespace myjob --create-namespace
   ```

3. Check status:
   ```bash
   kubectl get pods -n myjob
   kubectl get ingress -n myjob
   ```

4. Access:
   - Frontend: https://192.168.2.102
   - Grafana: http://192.168.2.102:30001

## Upgrade

```bash
helm upgrade myjob ./k8s/helm/myjob --namespace myjob
```

## Uninstall

```bash
helm uninstall myjob --namespace myjob
kubectl delete namespace myjob
kubectl delete namespace monitoring
```

## Architecture

```
k3s (Traefik ingress built-in)
├── Namespace: myjob
│   ├── PostgreSQL (StatefulSet)
│   ├── Redis (StatefulSet)
│   ├── LiveKit (Deployment)
│   ├── API (Deployment)
│   ├── Worker (Deployment)
│   ├── Browser Agent (Deployment)
│   └── Frontend (Deployment)
└── Namespace: monitoring
    ├── Prometheus (Deployment + PVC)
    └── Grafana (Deployment)
```

## Resource Budget

Total: ~2.1GB requests, ~3.3GB limits (fits in 4GB RAM)
