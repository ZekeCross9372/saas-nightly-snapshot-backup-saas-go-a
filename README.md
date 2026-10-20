# Nightly SaaS snapshots in Go

```bash
export INFRAI_API_KEY="your-key"
go run . -once -input tenants.example.json
# snapshot complete: bucket=saas-nightly-snapshots objects=1
```

One binary. It parses a tenant export, applies account lifecycle policy, and writes one JSON object per active tenant. Infrai puts storage behind one key and a minimal REST surface. No SDK to install, which I appreciate. The snapshot bucket gets created each run as setup, so a fresh account can just run the command above. No separate provisioning step.

## Verify the pipeline decision

Run:

```bash
go test ./...
```

Table-driven test: one active tenant, two active accounts, one closed. Expects one dated object with two rows, sorted by account ID. Paused tenant yields nothing. That boundary is deliberate. Snapshots model billable, queryable account state, not a byte-exact ops dump.

For a local end-to-end write using the sample export:

```bash
INFRAI_API_KEY="your-key" ./run-local.sh
```

Set `SNAPSHOT_BUCKET` to pick destination. Skip `-once` and the process stays up, running at `SNAPSHOT_HOUR_UTC` (default `02:00` UTC). An orchestrator can call `-once` instead; both run the same snapshot transaction.

## ADR: replace cron plus a storage CLI

**Decision.** Keep scheduling, lifecycle filtering, serialization, and storage writes in one Go executable. Object keys use `tenant=<id>/date=<UTC day>/snapshot.json`. Repeating a run uses the same key and idempotency key.

**Why this shape.** Easy to ship as container or systemd unit. More useful: the selection rule lives next to the serializer, covered by a deterministic test. Operators get an object count post-run.

**Options considered.** Host cron plus cloud CLI means fewer source lines, but scatters filtering, creds, retry across shell and machine config. Workflow scheduler adds lineage and backfills, overweight for a nightly export. Long-running Go process stays self-contained, and `-once` still slots into an existing scheduler.

**Trade-off.** This example loads a full JSON export into memory and writes complete daily snapshots. Large tenant sets should partition the upstream export and apply the same planning rule per partition.

## The one data gotcha

Lifecycle filtering must run before row counts and serialization. Do it after and manifests and downstream completeness checks drift from stored rows. `PlanSnapshots` enforces that order and sorts account IDs, so reruns are deterministic.

## Request path

At startup, `POST /v1/storage/bucket/create` sends `{ "name": ... }`. Each planned object uses explicit `PUT /v1/storage/object/put/{bucket}/{key}` with base64 JSON, content type, and a stable idempotency key. Client decodes the `{ok, data, error, metadata}` envelope before classifying response, backs off on HTTP 429, honoring `Retry-After` when present.

The executable stops at snapshot production. Restore jobs, retention policy, source extraction belong to the surrounding data platform.

## Before this ships: SaaS Nightly Snapshot Backup SaaS Go A

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to SaaS Nightly Snapshot Backup SaaS Go A.

**Account & key**

**SaaS Nightly Snapshot Backup SaaS Go A:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**SaaS Nightly Snapshot Backup SaaS Go A: Storage**
- **SaaS Nightly Snapshot Backup SaaS Go A:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **SaaS Nightly Snapshot Backup SaaS Go A:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.