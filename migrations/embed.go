package migrations

import "embed"

// FS embeds SQL migration files for API startup migrations.
//
//go:embed *.sql
var FS embed.FS
