# K3s Migration Design

## Overview

Migrate the MyJob application from Docker Compose to k3s (lightweight Kubernetes) for:
- Learning Kubernetes
- Production stability (auto-restart, self-healing, rolling updates)
- Resource efficiency
- Scalability (multi-node ready)

**Existing Helm charts:** `k8s/helm/myjob/` already has deployments, statefulsets, ingress, HPA, PDB, network policies, and service accounts. We're building on these, not starting from scratch.

## Constraints

- **Server:** Single-node home server at 192.168.2.102
- **RAM:** 4GB total (~3.5GB usable after k3s)
- **Storage:** HDD (large, slower)
- **Management:** Helm charts for templated deployments
- **Monitoring:** Prometheus + Grafana included

## Architecture

```
k3s (Traefik ingress built-in)
│
├── Namespace: myjob
│   ├── PostgreSQL (StatefulSet, 512MB limit)
│   ├── Redis (Deployment + PVC, 256MB limit)
│   ├── API (Deployment, 512MB limit)
│   ├── Worker (Deployment, 512MB limit)
│   ├── Browser Agent (Deployment, 768MB limit)
│   ├── Frontend (Deployment, 256MB limit)
│   └── LiveKit (Deployment, 256MB limit)
│
├── Namespace: monitoring
│   ├── Prometheus (StatefulSet, 256MB limit)
│   └── Grafana (Deployment, 128MB limit)
│
└── Ingress (Traefik)
    ├── 192.168.2.102 → Frontend
    ├── 192.168.2.102/api → API
    └── 192.168.2.102/livekit → LiveKit (WebSocket)
```

**Total RAM budget:** ~3.5GB used, ~500MB buffer for k3s system

## Helm Chart Structure

```
charts/myjob/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── namespace.yaml
│   ├── secrets.yaml
│   ├── postgres/
│   │   ├── statefulset.yaml
│   │   ├── service.yaml
│   │   └── pvc.yaml
│   ├── redis/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── pvc.yaml
│   ├── api/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── worker/
│   │   └── deployment.yaml
│   ├── browser-agent/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── frontend/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── livekit/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── ingress.yaml
```

## values.yaml (Complete)

```yaml
global:
  namespace: myjob
  imageRegistry: ghcr.io
  imagePrefix: anthrodjear

# ---- Secrets ----
secrets:
  postgresPassword: "CHANGE_ME_STRONG_PASSWORD_HERE"
  redisPassword: "CHANGE_ME_REDIS_PASSWORD_HERE"
  jwtSecret: "CHANGE_ME_64_CHAR_RANDOM_STRING_HERE"
  sessionSecret: "CHANGE_ME_ANOTHER_64_CHAR_RANDOM_STRING_HERE"
  livekitApiKey: "CHANGE_ME_LIVEKIT_KEY"
  livekitApiSecret: "CHANGE_ME_LIVEKIT_SECRET"
  openaiApiKey: ""
  anthropicApiKey: ""
  msTenantId: ""
  msClientId: ""
  msClientSecret: ""
  grafanaPassword: "CHANGE_ME_GRAFANA_PASSWORD"

# ---- PostgreSQL ----
postgres:
  image: pgvector/pgvector:pg16
  storage: 5Gi
  resources:
    limits:
      memory: 512Mi
    requests:
      memory: 256Mi
  maxOpenConns: 25
  maxIdleConns: 5

# ---- Redis ----
redis:
  image: redis:7-alpine
  storage: 1Gi
  resources:
    limits:
      memory: 256Mi
    requests:
      memory: 128Mi

# ---- API ----
api:
  image: myjob-api
  tag: latest
  replicas: 1
  resources:
    limits:
      memory: 512Mi
    requests:
      memory: 256Mi
  appEnv: production
  queueConcurrency: 10
  rateLimitRpm: 120
  rateLimitBurst: 20
  ollamaModel: "nemotron-3-super:cloud"
  ollamaEmbedModel: "granite-embedding:latest"
  corsOrigins: "http://localhost:3000,http://127.0.0.1:3000,http://192.168.2.102:3000,http://192.168.2.102"
  emailFolders: "Inbox,Jobs,Applications"

# ---- Worker ----
worker:
  image: myjob-worker
  tag: latest
  replicas: 1
  resources:
    limits:
      memory: 512Mi
    requests:
      memory: 256Mi
  queueConcurrency: 10

# ---- Browser Agent ----
browserAgent:
  image: myjob-browser-agent
  tag: latest
  replicas: 1
  resources:
    limits:
      memory: 768Mi
    requests:
      memory: 384Mi

# ---- Frontend ----
frontend:
  image: myjob-frontend
  tag: latest
  replicas: 1
  resources:
    limits:
      memory: 256Mi
    requests:
      memory: 128Mi
  nextPublicApiUrl: "/api"

# ---- LiveKit ----
livekit:
  image: livekit/livekit-server
  tag: v1.7.1
  resources:
    limits:
      memory: 256Mi
    requests:
      memory: 128Mi

# ---- Monitoring ----
monitoring:
  enabled: true
  prometheus:
    storage: 5Gi
    resources:
      limits:
        memory: 256Mi
  grafana:
    resources:
      limits:
        memory: 128Mi
```

## Ingress Configuration

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myjob-ingress
  namespace: myjob
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  tls:
    - hosts:
        - 192.168.2.102
      secretName: myjob-tls
  rules:
    - host: 192.168.2.102
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: api
                port:
                  number: 8080
          - path: /livekit
            pathType: Prefix
            backend:
              service:
                name: livekit
                port:
                  number: 7880
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 3000
```

## Secrets Management

Kubernetes Secrets mounted as environment variables:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: myjob-secrets
  namespace: myjob
type: Opaque
stringData:
  POSTGRES_PASSWORD: <value>
  REDIS_PASSWORD: <value>
  AUTH_JWT_SECRET: <value>
  SESSION_SECRET: <value>
  LIVEKIT_API_KEY: <value>
  LIVEKIT_API_SECRET: <value>
  OPENAI_API_KEY: <value>
  ANTHROPIC_API_KEY: <value>
  MS_TENANT_ID: <value>
  MS_CLIENT_ID: <value>
  MS_CLIENT_SECRET: <value>
```

Each deployment references the secret via `secretKeyRef`.

**Production upgrade:** Replace with `bitnami/sealed-secrets` or external secret store.

## Deployment Commands

```bash
# 1. Install k3s
curl -sfL https://get.k3s.io | sh -

# 2. Verify k3s
kubectl get nodes
kubectl get pods -A

# 3. Add Helm repo for monitoring
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# 4. Deploy Prometheus + Grafana
helm install monitoring prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.resources.limits.memory=256Mi \
  --set grafana.resources.limits.memory=128Mi \
  --set grafana.adminPassword=CHANGE_ME

# 5. Deploy MyJob
helm install myjob ./charts/myjob \
  --namespace myjob \
  --create-namespace \
  --set secrets.postgresPassword=YOUR_PASSWORD \
  --set secrets.jwtSecret=YOUR_SECRET

# 6. Check status
kubectl get pods -n myjob
kubectl get ingress -n myjob

# 7. Access
# Frontend: https://192.168.2.102
# Grafana:  http://192.168.2.102:3001 (NodePort or port-forward)
```

## Traffic Flow

```
Browser → https://192.168.2.102
         ↓ Traefik (TLS termination)
         ├── /api → api:8080
         ├── /livekit → livekit:7880 (WebSocket)
         └── / → frontend:3000
```

## Success Criteria

1. All services running in k3s
2. HTTPS accessible at https://192.168.2.102
3. Prometheus scraping metrics
4. Grafana dashboards accessible
5. Resource usage under 3.5GB total
6. Helm install/upgrade works
7. `kubectl get pods` shows all healthy

## Future Improvements

- Add `bitnami/sealed-secrets` for secret management
- Add cert-manager for automatic Let's Encrypt certificates
- Add PersistentVolumeClaims for PostgreSQL and Redis
- Add HorizontalPodAutoscaler for API/Worker
- Add network policies
- Add PodDisruptionBudgets
- Upgrade to multi-node cluster
