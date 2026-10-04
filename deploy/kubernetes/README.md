# Kubernetes Reference Baseline

A production-shaped [kustomize](https://kustomize.io) baseline for the Luas API, its workers and
retention jobs, and the Next.js Web application. It is an **example surface**: copy it into the
downstream deployment repository and own it there. Luas does not choose a cloud, ingress
controller, secret store, rollout controller, or PostgreSQL provider. Those choices stay with the
downstream team.

The container and runtime contract this baseline encodes is in
[`api/docs/DEPLOYMENT.md`](../../api/docs/DEPLOYMENT.md). Read that first; this README covers only
how the manifests map onto it.

## Layout

| Path | Contents |
|---|---|
| `config/` | `luas-api-config` ConfigMap: non-secret API runtime settings shared by every API pod, worker, and job |
| `base/` | API Deployment + Service + PDB, workflow worker, Web Deployment + Service + PDB |
| `migrate/` | Pre-deploy `luas migrate --force` Job, using the same image and configuration |
| `components/maintenance` | Retention CronJobs for authentication sessions and workflow tasks |
| `components/notification` | `notification:work` delivery worker |
| `components/webhook` | `webhook:work` delivery worker and `webhook:prune` retention job |
| `components/asset` | `asset:prune` cleanup job |
| `overlays/example` | Base plus `organization,notification,webhook` and their components |

Every pod runs non-root with a read-only root filesystem, all capabilities dropped,
`RuntimeDefault` seccomp, no service-account token, and an `emptyDir` for writable scratch space.
The API uses `/health/live` for liveness and `/health/ready` for readiness, so a database outage
takes replicas out of traffic without restarting them.

## Deploy

1. **Build and push images by digest.** Build `api/` and `web/` with their Dockerfiles, then set
   the digest in each kustomization's `images` entry (`kustomize edit set image
   luas-api=registry.example.com/luas-api@sha256:...`). The Web image bakes in `NEXT_PUBLIC_*`
   at build time, so build it once per environment origin.

2. **Create the secret outside this repository.** It needs at least `DB_USERNAME` and
   `DB_PASSWORD`, plus starter secrets when they are selected (`WEBHOOK_ENCRYPTION_KEY`,
   `ASSET_TRANSFER_SIGNING_KEY`, the R2 group, mail credentials). Prefer your platform's secret
   operator. The imperative equivalent:

   ```bash
   kubectl -n luas create secret generic luas-api-secrets --from-env-file=api.secrets.env
   ```

3. **Set the configuration.** Edit `config/config.yaml` or patch it from an overlay: `APP_URL`,
   `CORS_ALLOW_ORIGINS`, `DB_HOST`, `OPTIONAL_STARTERS`, and `SERVER_TRUSTED_PROXIES`. Set
   `SERVER_TRUSTED_PROXIES` to the pod CIDR that the Web pods and ingress use. Without it, the API
   logs and rate-limits the Web pod's address instead of the client's. Your ingress must also
   overwrite `X-Real-IP` on every request it forwards to Web.

4. **Migrate, then roll out.** Apply the migration job and wait for it to finish:

   ```bash
   kubectl -n luas apply -k deploy/kubernetes/migrate
   kubectl -n luas wait --for=condition=complete --timeout=10m job/luas-migrate
   ```

   Then roll out the overlay:

   ```bash
   kubectl -n luas apply -k deploy/kubernetes/overlays/example
   ```

   The migration job must use the same `OPTIONAL_STARTERS` as the serving pods.

5. **Serve the Admin Console separately.** `admin/` builds to static files for object storage and
   a CDN; see [`admin/docs/DEPLOYMENT.md`](../../admin/docs/DEPLOYMENT.md). Its origin goes in
   `OPERATOR_ALLOWED_ORIGINS` when the `operator` starter is selected.

## Adapting

- **Selecting starters.** Change `OPTIONAL_STARTERS` once, in the overlay's ConfigMap patch. Then
  add exactly the matching components. A component without its starter fails at start-up.
- **Scaling.** Workers use database leases, so replicas scale horizontally. Budget
  `DB_MAX_OPEN_CONNS` × (API + worker + job pods) against PostgreSQL or pooler capacity.
- **Things the baseline omits on purpose:** Ingress or Gateway, NetworkPolicy,
  HorizontalPodAutoscaler, and ServiceMonitor all depend on the cluster. Add them in your overlay.
  `GET /metrics` is served on the API port.

## Verify

```bash
kubectl kustomize deploy/kubernetes/overlays/example | kubeconform -strict -summary -kubernetes-version 1.30.0
```
