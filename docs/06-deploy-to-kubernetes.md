## Deploy to Kubernetes

The starter ships with Docker Compose for a five-minute local demo. For production you'll want Kubernetes (or Fly.io / Railway / Render — any container host works).

This doc describes the shape of a Kubernetes deploy. There's no Helm chart in the starter itself; adapt this to your cluster's conventions.

## Three pieces

1. **Postgres with pgvector** — a cluster (e.g. CloudNativePG) or a managed service (Aiven, Neon). Must have the `vector` extension installed. For self-hosted CloudNativePG, either build a custom image with `postgresql-<version>-pgvector` via apt, or use the official pgvector-enabled image where available.

2. **Ollama** — runs on a host with enough RAM/GPU for your chosen model. Often NOT in the Kubernetes cluster — a beefy VM (ideally with a GPU) talks back to the cluster over the network. Make sure it binds `0.0.0.0:11434`, not `127.0.0.1`:
   ```
   OLLAMA_HOST=0.0.0.0:11434
   ```
   Or use a managed inference provider instead.

3. **ragbot** — the Go service. Stateless, scale horizontally. One Deployment, one Service, one Ingress.

## Minimal manifest sketch

```yaml
apiVersion: apps/v1
kind: Deployment
metadata: {name: ragbot, namespace: default}
spec:
  replicas: 2
  selector: {matchLabels: {app: ragbot}}
  template:
    metadata: {labels: {app: ragbot}}
    spec:
      containers:
      - name: ragbot
        image: your-registry/ragbot:v1.0.0
        ports: [{containerPort: 8080, name: http}]
        env:
        - name: DB_DSN
          valueFrom: {secretKeyRef: {name: ragbot-db, key: dsn}}
        - name: OLLAMA_URL
          value: http://10.0.1.120:11434
        - name: OLLAMA_CHAT_MODEL
          value: gemma3:4b
        - name: OLLAMA_EMBED_MODEL
          value: nomic-embed-text
        - name: PRODUCT_NAME
          value: Acme
        - name: PRODUCT_BLURB
          value: an invoicing SaaS for freelancers.
        livenessProbe:  {httpGet: {path: /healthz, port: http}}
        readinessProbe: {httpGet: {path: /healthz, port: http}}
        resources:
          requests: {cpu: 100m, memory: 256Mi}
          limits:   {cpu: 500m, memory: 512Mi}
---
apiVersion: v1
kind: Service
metadata: {name: ragbot, namespace: default}
spec:
  selector: {app: ragbot}
  ports: [{port: 80, targetPort: http}]
```

Then add your usual Ingress / IngressRoute pointing at the Service.

## Critical config

- **`DB_DSN`** should live in a Secret (External Secrets Operator + Vault is the clean path).
- **`OLLAMA_URL`** points at wherever your Ollama runs. If the Ollama host is outside the cluster, make sure the cluster can reach it (network policies, VPC peering).
- **Streaming responses**: if you have an Ingress that buffers by default, disable it. For Nginx set `proxy_buffering off`; for Traefik it works out of the box; for Cloudflare, text/event-stream is handled correctly automatically.

## Gotchas specific to K8s deployments

Most of the generic gotchas are captured in `07-troubleshooting.md`. A few that only bite once you run on Kubernetes:

- Your service's `http.Server` must set `WriteTimeout: 0` (or very long). The default 10s truncates SSE streams mid-answer.
- CiliumNetworkPolicy default-deny → egress to the Postgres cluster and Ollama must be explicitly allowed.
- If you have a shared middleware chain that wraps `http.ResponseWriter`, make sure every wrapper exposes `Flush()` for SSE to work.

## Observability

The starter emits basic logs and exposes `/healthz`. Add:

- **Tracing** — wrap the Ollama HTTP client with OTel-instrumented transport, export to Tempo / Jaeger.
- **Metrics** — `/metrics` endpoint via `github.com/prometheus/client_golang`. Count queries per second, measure p95 response time, track corpus-reindex duration.
- **Feedback loop** — the bot already persists thumbs up/down in `ragbot_messages.feedback`. Wire an alert on sustained 👎 rates to know when your docs need work.

## Cost at production scale

For a widget on a marketing site doing 10k queries/day:

- **Postgres**: smallest managed instance (e.g. 1 vCPU, 4 GB, 10 GB storage) — $15/month.
- **Ollama host**: a mid-range GPU VM (e.g. RTX 4060 Ti, 16 GB VRAM) — $100/month, or use a managed API.
- **ragbot**: tiny, fits on a $5 container host.

Total: ~$30-150/month depending on LLM choice. Compare to $1000+/month for Intercom's AI features and you see why self-hosting is attractive.
