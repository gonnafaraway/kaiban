package migrations

import "embed"

//go:embed api/*.sql
var FS embed.FS
