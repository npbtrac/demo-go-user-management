# User management microservice

Go service for user CRUD, backed by Postgres. Auth is out of scope.

This implements [issue #1](https://github.com/npbtrac/demo-go-user-management/issues/1).

## Which surface to use

| Surface | Address (local Docker) | Access | Scope | Docs |
| --- | --- | --- | --- | --- |
| gRPC `usermgmt.v1.UserService` | `localhost:10102` | Internal mesh only | Full CRUD | [Proto](api/proto/README.md), [user.proto](api/proto/usermgmt/v1/user.proto) |
| Internal REST | `http://localhost:10101` | Internal services | Full CRUD | [OpenAPI](api/openapi/internal.yaml) |
| Public REST | `http://localhost:10100` | External clients | `GET /public/v1/users/{id}` only | [OpenAPI](api/openapi/public.yaml) |

Bind the three listeners on separate ports so they can be firewalled independently. Do not expose gRPC or `/internal` publicly.

## Local development

Two options; do not run the Docker app and `make dev` at the same time (same ports).

### App in Docker

```bash
make compose-up         # start Postgres + app
make compose-rebuild    # rebuild the app image and recreate containers
make compose-build      # rebuild images only
make compose-down       # stop and remove containers
```

Or: `docker compose up -d`. Code changes are **not** picked up until `make compose-rebuild`.

### App on the host (Air live-reload)

```bash
make compose-up-db      # Postgres only
make dev                # Air watches .go files and restarts the process
```


The process runs embedded migrations against Postgres on startup (same SQL as tests: `db/migrations`).

| Variable | Local default | Meaning |
| --- | --- | --- |
| `PUBLIC_HTTP_PORT` | `10100` | Host / public REST port |
| `INTERNAL_HTTP_PORT` | `10101` | Host / internal REST port |
| `GRPC_PORT` | `10102` | Host / gRPC port |
| `POSTGRES_PORT` | `10112` | Host port mapped to Postgres `5432` in the container |
| `DATABASE_URL` | required | Postgres URL for the Go process. Compose overrides this to `postgres:5432` on the Docker network. On the host use `localhost:10112`. |
| `PUBLIC_HTTP_ADDR` | `:10100` | Public REST listen address |
| `INTERNAL_HTTP_ADDR` | `:10101` | Internal REST listen address |
| `GRPC_ADDR` | `:10102` | gRPC listen address |

Health checks: `GET http://localhost:10100/healthz` and `GET http://localhost:10101/healthz`.

Against a running stack:

```bash
make smoke              # public REST + internal REST + gRPC
make smoke-public
make smoke-internal
make smoke-grpc
```

## API examples

Local ports come from `.env` (`10100` public, `10101` internal, `10102` gRPC). Copy `id` from the create response into the get calls.

Create a user (internal REST):

```bash
curl -sS -X POST http://localhost:10101/internal/v1/users \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "phone": "+15551212",
    "params": {"plan": "pro"}
  }'
```

View a user (internal REST):

```bash
curl -sS http://localhost:10101/internal/v1/users/<USER_ID>
```

View a user (public REST):

```bash
curl -sS http://localhost:10100/public/v1/users/<USER_ID>
```

View a user (gRPC). Use [grpcurl](https://github.com/fullstorydev/grpcurl); the server exposes reflection:

```bash
grpcurl -plaintext \
  -d '{"id":"<USER_ID>"}' \
  localhost:10102 \
  usermgmt.v1.UserService/GetUser
```

Create a user (gRPC):

```bash
grpcurl -plaintext \
  -d '{"username":"bob","email":"bob@example.com","phone":"+1","params":{"plan":"pro"}}' \
  localhost:10102 \
  usermgmt.v1.UserService/CreateUser
```

List RPCs:

```bash
grpcurl -plaintext localhost:10102 list
```


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
