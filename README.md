# base-microservice

Reusable Go + React monorepo template. Drop a new
bounded context in, rename a few placeholders, and you have a working
Connect-RPC service + BFF + SPA + K8s manifests on day one.

The template ships with one example domain, **`user`** (register / login
/ list), end-to-end:

```
proto/user/*.proto            ─┐
                                ├─→ gen/user/  (Go server+client)
                                └─→ apps/webapp/web/src/gen/  (TS client)

services/user/                 backend service (Postgres + JetStream + JWT)
apps/webapp/cmd/webapp/        BFF (cookie session → forward-auth headers)
apps/webapp/web/               React 19 + TS + Tailwind 4 SPA
```

## Stack

| Layer    | Choice |
| -------- | ------ |
| Language | Go 1.25+, TypeScript 5 |
| RPC      | [Connect-RPC](https://connectrpc.com) over HTTP/2 (h2c) + Buf |
| Database | PostgreSQL 16 — one DB per service |
| ORM      | GORM (AutoMigrate in dev; `migrations/*.sql` for prod) |
| Events   | NATS JetStream, single stream `base`, subjects `base.<ctx>.v1.<Event>` |
| Auth     | JWT (HS256) + HttpOnly cookies. BFF authenticates, services trust forward-auth headers behind ingress. |
| Frontend | React 19, react-router 7, Tailwind 4, Connect-Web |
| Build    | docker buildx bake, distroless Go + Caddy for SPA |
| Deploy   | kustomize, overlays per env |
| Toolchain| `mise` (pinned versions), `pnpm` workspace, `overmind` dev runner |

## Layout

```
.
├── proto/                        proto contracts
│   ├── user/                     backend service + events
│   └── apps/webapp/v1/api.proto  BFF wire contract
├── gen/                          generated Go (commit if you do)
├── pkg/                          shared Go libs (each a module)
│   ├── actor   — forward-auth header conventions
│   ├── auth    — JWT issuer/verifier + bcrypt
│   ├── config  — Kong wrapper (env prefix BASE_<SVC>_)
│   ├── db      — gorm.Open + pool tuning
│   ├── nats    — JetStream connect + stream/consumer helpers
│   ├── obs     — zerolog + Connect logging interceptor
│   └── outbox  — transactional outbox writer
├── services/
│   └── user/                     backend service (CRUD + events)
├── apps/
│   └── webapp/                   BFF (cmd/) + SPA (web/)
├── packages/                     shared FE TS packages
│   ├── ui              — primitive components (workspace:*)
│   ├── shared-auth     — role helpers
│   └── eslint-config
├── deploy/k8s/                   kustomize base + overlays/dev|prod
├── build/                        generic Dockerfiles (go.Dockerfile, web.Dockerfile)
├── docker/postgres/init.sql      per-service DB bootstrap
├── docker-compose.yml            local infra (postgres + nats)
├── docker-bake.hcl               image targets
├── Makefile, Procfile.dev, .mise.toml, lefthook.yml, buf.yaml
└── go.work, pnpm-workspace.yaml, tsconfig.base.json
```

## Quick start

### 1. Install tooling

```bash
# mise pins Go, Node, pnpm, buf, golangci-lint, lefthook, overmind
curl https://mise.run | sh
mise install

cp .env.example .env
```

### 2. Bring up infra

```bash
make infra-up           # postgres + nats via docker-compose
```

### 3. Generate code from proto

```bash
make proto              # Go (gen/) + TS (apps/webapp/web/src/gen/)
```

### 4. Install JS deps + tidy Go modules

```bash
pnpm install
make tidy
```

### 5. Run everything

```bash
make dev                # overmind: user service + BFF + Vite SPA
# or, in three terminals:
make dev-user
make dev-webapp-bff
make dev-webapp-web
```

Open <http://localhost:3000>, register a user, you're in.

## Adding a new service

1. Copy `services/user` → `services/<newctx>`. Update `go.mod` module path
   + replace blocks. Add the directory to `go.work`.
2. Copy `proto/user/*.proto` → `proto/<newctx>/`. Update `package` and
   `go_package` options. Run `make proto`.
3. Create the database in `docker/postgres/init.sql` and (if needed)
   `deploy/k8s/base/postgres/statefulset.yaml`'s init ConfigMap.
4. Add Kustomize folder under `deploy/k8s/base/services/<newctx>/` (copy
   `user/` and rename).
5. Add a `docker-bake.hcl` target.
6. Wire the new backend client into the BFF (`apps/webapp/internal/clients.go`)
   when the SPA needs it.

## Adding a new app (BFF + SPA)

1. Copy `apps/webapp` → `apps/<newapp>`. Update `go.mod`, `package.json`
   names, Vite proxy + port.
2. Copy `proto/apps/webapp/v1/api.proto` → `proto/apps/<newapp>/v1/api.proto`.
3. `apps/<newapp>/web/buf.gen.yaml` already only targets its own proto file,
   so the FE bundle stays small.
4. Each app owns its auth slice — keep cookie names unique (`_<app>_access`).

## Auth model

- **User credentials** live in `services/user`. `Login` returns access +
  refresh JWTs signed with the shared `JWT_SECRET`.
- **BFF** receives the JWTs and writes them as HttpOnly cookies
  (`_base_access`, `_base_refresh`).
- For each browser request, the BFF verifies the access cookie and
  forwards `X-Forwarded-User`/`X-Forwarded-Email`/`X-Forwarded-Roles` to
  backend services via outbound Connect interceptors.
- **Backends** trust those headers (`pkg/actor.ConnectInterceptor`). They
  MUST be stripped at ingress.
- For production, swap HS256 for RS256/EdDSA and serve a JWKS endpoint
  from the user service.

## Events / outbox

`pkg/outbox` ships only the writer. Domain code does, inside a tx:

```go
outbox.Enqueue(tx, &outbox.Event{
    AggregateID: u.ID, EventType: "UserRegistered",
    Subject: "base.user.v1.UserRegistered", MsgID: ...,
    Payload: payloadBytes,
})
```

A dispatcher (TBD — copy `apps/outbox-dispatcher` from xcap-v3 when you
need at-least-once) drains `outbox_events` and publishes onto JetStream.

For simple use cases, the example service publishes directly
(`services/user/internal/consumer/publisher.go`) — fine for dev, but
loses atomicity with the DB write.

## Deploy

```bash
make images TAG=v0.1.0          # builds all images
make k8s-dev                    # apply dev overlay
make k8s-prod                   # apply prod overlay (after editing image registry)
```

Replace the placeholder `replace-me-with-32-plus-char-random-string`
secrets with real values (Sealed Secrets / External Secrets recommended).

## Conventions worth keeping

- **Multi-module Go workspace.** Each service/app/pkg has its own
  `go.mod` + `replace` against `../../gen` and the pkgs it uses. `go.work`
  glues them together for dev; consumers outside the workspace still
  resolve.
- **Per-service Postgres database.** Never share schemas.
- **`Nats-Msg-Id` for dedup.** Use `<event>:<aggregate>:<version>`.
- **Forward-auth, not JWT-in-backend.** Backends trust headers; only the
  BFF parses tokens. Simpler + faster.
- **Generated code is committed** if you don't want every clone to run
  `make proto`. Decision is yours — the `gen/` README explains both.
- **AutoMigrate flag for dev only.** Prod uses `golang-migrate`.

## What's intentionally NOT here

Compared to xcap-v3, this template drops:

- Casdoor OAuth (→ JWT)
- Casbin role enforcement (TODO: add a simple role-check middleware)
- Per-FE shared `i18n` (add when you actually need multi-locale)
- Outbox dispatcher binary (writer only — copy when needed)
- Flipt / SigNoz platform stack
- Playwright e2e harness

Add them back deliberately when the project requires them — this base
stays minimal so it's actually reusable.
