# Sprint 2 — Make it Deliverable 🚀

ครบข้อมูลแล้วครับ ลุยเลย! ใช้ stack: **Woodpecker 3.13 → Docker Hub → k3s + nginx-ingress**

---

## 📁 Files to Create

```
DevOpsAgents/
├── .woodpecker.yml                    ← NEW (CI/CD)
├── backend/
│   ├── Dockerfile                     ← NEW
│   └── .dockerignore                  ← NEW
├── frontend/
│   ├── Dockerfile                     ← NEW
│   ├── .dockerignore                  ← NEW
│   └── next.config.mjs                ← UPDATE (standalone)
└── deploy/
    └── k8s/
        ├── 00-namespace.yaml          ← NEW
        ├── 01-secrets.yaml.example    ← NEW
        ├── 10-backend-pvc.yaml        ← NEW
        ├── 11-backend.yaml            ← NEW
        ├── 12-frontend.yaml           ← NEW
        └── 20-ingress.yaml            ← NEW
```

---

## 🐳 1. Backend Dockerfile

### `backend/Dockerfile`
```dockerfile
# syntax=docker/dockerfile:1.7

# ── Builder ─────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOTOOLCHAIN=local

# Cache deps layer
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/server .

# ── Runtime (distroless, ~2MB base) ─────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/server /app/server

# SQLite จะถูกเขียนที่ /data (mount เป็น PVC)
ENV DB_PATH=/data/auth.db \
    PORT=8080

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
```

### `backend/.dockerignore`
```
*.db
*.db-journal
auth.db
coverage.out
internal/**/*_test.go
.git
.gitignore
README.md
```

---

## 🐳 2. Frontend Dockerfile (Next.js standalone)

### `frontend/next.config.mjs` (UPDATE)
```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: "standalone",   // ← เพิ่มบรรทัดนี้
};

export default nextConfig;
```

### `frontend/Dockerfile`
```dockerfile
# syntax=docker/dockerfile:1.7

# ── Deps & Build (Bun) ──────────────────────────────────────────
FROM oven/bun:1 AS builder

WORKDIR /app

# baked-in build-time env (Next.js public envs)
ARG NEXT_PUBLIC_API_URL=""
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_TELEMETRY_DISABLED=1

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .
RUN bun run build

# ── Runtime (small Node.js) ─────────────────────────────────────
FROM node:20-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

RUN addgroup -S -g 1001 nodejs && adduser -S -u 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]
```

### `frontend/.dockerignore`
```
node_modules
.next
out
__tests__
coverage
.env*
!.env.example
.git
README.md
```

---

## ☸️ 3. Kubernetes Manifests (IaC)

### `deploy/k8s/00-namespace.yaml`
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: devopsagents
  labels:
    app.kubernetes.io/part-of: devopsagents
```

### `deploy/k8s/01-secrets.yaml.example`
> ⚠️ คัดลอกเป็น `01-secrets.yaml` แล้วใส่ค่าจริง — **อย่า commit ไฟล์ secret จริงเข้า git**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backend-secret
  namespace: devopsagents
type: Opaque
stringData:
  # สร้างด้วย: openssl rand -base64 48
  JWT_SECRET: "REPLACE_ME_WITH_A_STRONG_RANDOM_STRING"
```

### `deploy/k8s/10-backend-pvc.yaml`
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: backend-data
  namespace: devopsagents
spec:
  accessModes: [ReadWriteOnce]
  storageClassName: local-path   # k3s default
  resources:
    requests:
      storage: 1Gi
```

### `deploy/k8s/11-backend.yaml`
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  namespace: devopsagents
  labels: { app: backend }
spec:
  replicas: 1                    # SQLite RWO → 1 replica เท่านั้น
  strategy:
    type: Recreate               # เพื่อไม่ให้แย่ง mount PVC
  selector:
    matchLabels: { app: backend }
  template:
    metadata:
      labels: { app: backend }
    spec:
      containers:
        - name: backend
          image: akawatmor/devopsagents-backend:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 8080
          env:
            - name: PORT
              value: "8080"
            - name: DB_PATH
              value: /data/auth.db
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: backend-secret
                  key: JWT_SECRET
          volumeMounts:
            - name: data
              mountPath: /data
          readinessProbe:
            httpGet: { path: /api/health, port: 8080 }
            initialDelaySeconds: 3
            periodSeconds: 5
          livenessProbe:
            httpGet: { path: /api/health, port: 8080 }
            initialDelaySeconds: 10
            periodSeconds: 15
          resources:
            requests: { cpu: "50m", memory: "32Mi" }
            limits:   { cpu: "500m", memory: "128Mi" }
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: backend-data
---
apiVersion: v1
kind: Service
metadata:
  name: backend
  namespace: devopsagents
spec:
  selector: { app: backend }
  ports:
    - port: 8080
      targetPort: 8080
      name: http
```

### `deploy/k8s/12-frontend.yaml`
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: devopsagents
  labels: { app: frontend }
spec:
  replicas: 2
  selector:
    matchLabels: { app: frontend }
  template:
    metadata:
      labels: { app: frontend }
    spec:
      containers:
        - name: frontend
          image: akawatmor/devopsagents-frontend:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 3000
          env:
            - name: PORT
              value: "3000"
            - name: HOSTNAME
              value: "0.0.0.0"
          readinessProbe:
            tcpSocket: { port: 3000 }
            initialDelaySeconds: 3
            periodSeconds: 5
          resources:
            requests: { cpu: "50m", memory: "64Mi" }
            limits:   { cpu: "500m", memory: "256Mi" }
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
  namespace: devopsagents
spec:
  selector: { app: frontend }
  ports:
    - port: 3000
      targetPort: 3000
      name: http
```

### `deploy/k8s/20-ingress.yaml`
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: devopsagents
  namespace: devopsagents
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "5m"
    # ── ถ้าใช้ cert-manager ปลด comment ──
    # cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  rules:
    - host: devopsagent.akawatmor.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: backend
                port: { number: 8080 }
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port: { number: 3000 }
  # ── ถ้ามี TLS ──
  # tls:
  #   - hosts: [devopsagent.akawatmor.com]
  #     secretName: devopsagents-tls
```

---

## 🦝 4. Woodpecker CI/CD Pipeline

### `.woodpecker.yml`
```yaml
# Woodpecker 3.x syntax
when:
  - event: [push, pull_request]
    branch: [main, develop]
  - event: manual

variables:
  - &dh_user { from_secret: dockerhub_username }
  - &dh_pass { from_secret: dockerhub_token }

steps:
  # ─────────────────────────────────────────────────────────────
  # 1) Test backend
  # ─────────────────────────────────────────────────────────────
  test-backend:
    image: golang:1.25-alpine
    environment:
      GOTOOLCHAIN: local
    commands:
      - apk add --no-cache git build-base
      - cd backend
      - go vet ./...
      - go test -race -v ./...

  # ─────────────────────────────────────────────────────────────
  # 2) Test frontend
  # ─────────────────────────────────────────────────────────────
  test-frontend:
    image: oven/bun:1
    commands:
      - cd frontend
      - bun install --frozen-lockfile
      - bunx tsc --noEmit
      - bun test

  # ─────────────────────────────────────────────────────────────
  # 3) Build & push backend image (เฉพาะ main)
  # ─────────────────────────────────────────────────────────────
  build-backend:
    image: woodpeckerci/plugin-docker-buildx
    settings:
      repo: akawatmor/devopsagents-backend
      dockerfile: backend/Dockerfile
      context: backend
      platforms: linux/amd64
      tags:
        - latest
        - "${CI_COMMIT_SHA:0:8}"
      username: *dh_user
      password: *dh_pass
    depends_on: [test-backend]
    when:
      - event: push
        branch: main

  # ─────────────────────────────────────────────────────────────
  # 4) Build & push frontend image (เฉพาะ main)
  # ─────────────────────────────────────────────────────────────
  build-frontend:
    image: woodpeckerci/plugin-docker-buildx
    settings:
      repo: akawatmor/devopsagents-frontend
      dockerfile: frontend/Dockerfile
      context: frontend
      platforms: linux/amd64
      tags:
        - latest
        - "${CI_COMMIT_SHA:0:8}"
      build_args:
        - NEXT_PUBLIC_API_URL=https://devopsagent.akawatmor.com
      username: *dh_user
      password: *dh_pass
    depends_on: [test-frontend]
    when:
      - event: push
        branch: main

  # ─────────────────────────────────────────────────────────────
  # 5) Deploy to k3s (apply manifests + rolling update)
  # ─────────────────────────────────────────────────────────────
  deploy:
    image: bitnami/kubectl:1.30
    environment:
      KUBECONFIG_CONTENT:
        from_secret: kubeconfig
      IMAGE_TAG: "${CI_COMMIT_SHA:0:8}"
    commands:
      - mkdir -p $HOME/.kube
      - echo "$KUBECONFIG_CONTENT" > $HOME/.kube/config
      - chmod 600 $HOME/.kube/config

      # 5.1) Apply all non-secret manifests (idempotent IaC)
      - kubectl apply -f deploy/k8s/00-namespace.yaml
      - kubectl apply -f deploy/k8s/10-backend-pvc.yaml
      - kubectl apply -f deploy/k8s/11-backend.yaml
      - kubectl apply -f deploy/k8s/12-frontend.yaml
      - kubectl apply -f deploy/k8s/20-ingress.yaml

      # 5.2) Pin to current commit SHA (rolling update)
      - kubectl -n devopsagents set image deployment/backend  backend=akawatmor/devopsagents-backend:$IMAGE_TAG
      - kubectl -n devopsagents set image deployment/frontend frontend=akawatmor/devopsagents-frontend:$IMAGE_TAG

      # 5.3) Wait for rollout
      - kubectl -n devopsagents rollout status deployment/backend  --timeout=120s
      - kubectl -n devopsagents rollout status deployment/frontend --timeout=120s

      # 5.4) Smoke test
      - kubectl -n devopsagents get pods,svc,ingress
    depends_on: [build-backend, build-frontend]
    when:
      - event: push
        branch: main
```

---

## 🔐 5. Setup Steps (One-time)

### A) สร้าง Docker Hub Token
1. ไปที่ https://hub.docker.com/settings/security → **New Access Token** (`Read & Write`)
2. คัดลอก token ไว้

### B) เพิ่ม Secrets ใน Woodpecker
ที่ `https://woodpecker-kps.akawatmor.com` → เลือก repo → **Settings → Secrets → Add**

| Name | Value | Events |
|---|---|---|
| `dockerhub_username` | `akawatmor` | `push` |
| `dockerhub_token` | `dckr_pat_xxx...` | `push` |
| `kubeconfig` | เนื้อหาทั้งไฟล์ `~/.kube/config` | `push` |

> 💡 สำหรับ `kubeconfig`: ดึงมาจาก k3s ด้วย `sudo cat /etc/rancher/k3s/k3s.yaml` แล้วเปลี่ยน `server: https://127.0.0.1:6443` เป็น public IP ของ k3s

### C) สร้าง Secret บน cluster (รันครั้งเดียว ด้วยมือ)
```bash
# สร้าง random JWT secret + apply
JWT_SECRET=$(openssl rand -base64 48)

kubectl create namespace devopsagents --dry-run=client -o yaml | kubectl apply -f -

kubectl -n devopsagents create secret generic backend-secret \
  --from-literal=JWT_SECRET="$JWT_SECRET" \
  --dry-run=client -o yaml | kubectl apply -f -
```

### D) ตั้ง DNS
A record: `devopsagent.akawatmor.com` → IP ของ k3s node (port 80/443)

---

## ✅ 6. Manual First Deploy (ก่อน CI ทำงานครั้งแรก)

```bash
cd DevOpsAgents

# 1) build & push images ครั้งแรก
docker login
docker buildx build --platform linux/amd64 -t akawatmor/devopsagents-backend:latest  -f backend/Dockerfile  backend  --push
docker buildx build --platform linux/amd64 -t akawatmor/devopsagents-frontend:latest -f frontend/Dockerfile frontend \
  --build-arg NEXT_PUBLIC_API_URL=https://devopsagent.akawatmor.com --push

# 2) apply manifests
kubectl apply -f deploy/k8s/00-namespace.yaml
# (สร้าง secret backend-secret ตามขั้นตอน C ข้างบน)
kubectl apply -f deploy/k8s/10-backend-pvc.yaml
kubectl apply -f deploy/k8s/11-backend.yaml
kubectl apply -f deploy/k8s/12-frontend.yaml
kubectl apply -f deploy/k8s/20-ingress.yaml

# 3) verify
kubectl -n devopsagents get pods,svc,ingress
curl https://devopsagent.akawatmor.com/api/health
```

---

## 📊 Pipeline Flow

```
┌──────────────┐     ┌──────────────┐
│ test-backend │     │ test-frontend│        ← parallel
└──────┬───────┘     └──────┬───────┘
       │                    │
       ▼                    ▼
┌──────────────┐     ┌──────────────┐
│ build-backend│     │build-frontend│        ← parallel (push only)
└──────┬───────┘     └──────┬───────┘
       │                    │
       └─────────┬──────────┘
                 ▼
         ┌───────────────┐
         │    deploy     │                    ← apply manifests + rollout
         │ (k3s/kubectl) │
         └───────────────┘
```

---

## 🎯 Sprint 2 Checklist

| Requirement | Implementation |
|---|---|
| **CI/CD Pipeline** | `.woodpecker.yml` (test → build → push → deploy) |
| **Containers** | Multi-stage Dockerfile ทั้ง 2 service + distroless/alpine runtime |
| **IaC** | `deploy/k8s/*.yaml` — declarative, idempotent, version-controlled |
| Container registry | Docker Hub (`akawatmor/devopsagents-*`) |
| Orchestrator | k3s (single node) |
| Ingress | nginx → `devopsagent.akawatmor.com` |
| Storage | PVC (`local-path`, 1Gi) สำหรับ SQLite |
| Secrets | k8s Secret + Woodpecker secrets |

---

## 🚀 ขั้นตอนถัดไป (Sprint 3 ideas)

- **HTTPS** ด้วย cert-manager + Let's Encrypt
- **Observability**: Prometheus + Grafana / Loki
- **GitOps**: ArgoCD/FluxCD แทน `kubectl apply` ตรง
- **Migration** จาก SQLite → PostgreSQL (เปิดทาง multi-replica)
- **HPA** auto-scale frontend

---

