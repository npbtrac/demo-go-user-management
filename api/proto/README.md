# usermgmt.v1.UserService

Internal gRPC contract for user CRUD. Generated Go code lives next to [user.proto](usermgmt/v1/user.proto).

Regenerate:

```bash
buf dep update
buf generate
```

## RPCs

| RPC | Request | Response |
| --- | --- | --- |
| `CreateUser` | username, email, optional phone and params | `User` |
| `GetUser` | id | `User` |
| `ListUsers` | page_size, page_token | users + next_page_token |
| `UpdateUser` | id + optional fields | `User` |
| `DeleteUser` | id | Empty |

Error codes: `INVALID_ARGUMENT`, `NOT_FOUND`, `ALREADY_EXISTS`.
