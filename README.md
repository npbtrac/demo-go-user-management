# User management microservice

Go service for user CRUD, backed by Postgres. Auth is out of scope.

This implements [issue #1](https://github.com/npbtrac/demo-go-user-management/issues/1).

## Which surface to use

| Surface | Address (local Docker) | Access | Scope | Docs |
| --- | --- | --- | --- | --- |
| gRPC `usermgmt.v1.UserService` | `localhost:9090` | Internal mesh only | Full CRUD | [Proto](api/proto/README.md), [user.proto](api/proto/usermgmt/v1/user.proto) |
| Internal REST | `http://localhost:8081` | Internal services | Full CRUD | [OpenAPI](api/openapi/internal.yaml) |
| Public REST | `http://localhost:8080` | External clients | `GET /public/v1/users/{id}` only | [OpenAPI](api/openapi/public.yaml) |

Bind the three listeners on separate ports so they can be firewalled independently. Do not expose gRPC or `/internal` publicly.

## Local Docker

```bash
docker compose -f deploy/docker/docker-compose.yml up --build
```

The process runs embedded migrations against Postgres on startup (same SQL as tests: `db/migrations`).

| Variable | Default | Meaning |
| --- | --- | --- |
| `DATABASE_URL` | required | Postgres URL, e.g. `postgres://usermgmt:usermgmt@localhost:5432/usermgmt?sslmode=disable` |
| `PUBLIC_HTTP_ADDR` | `:8080` | Public REST listen address |
| `INTERNAL_HTTP_ADDR` | `:8081` | Internal REST listen address |
| `GRPC_ADDR` | `:9090` | gRPC listen address |

Health checks: `GET http://localhost:8080/healthz` and `GET http://localhost:8081/healthz`.

## Run tests

```bash
go test ./...
```

Unit tests use an in-memory repository. Integration tests start Postgres with Testcontainers when Docker is available, or use `TEST_DATABASE_URL` if set.

Regenerate gRPC stubs after proto changes:

```bash
buf dep update
buf generate
```
