package migrations

import "embed"

// FS contains SQL migration files shared by local Docker, the service, and tests.
//
//go:embed *.sql
var FS embed.FS
