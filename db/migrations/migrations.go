// Package migrations embeds the SQL migration files into the binary.
package migrations

import "embed"

// FS holds every migration file under db/migrations/*.sql.
//
// One Go process serves the SPA, REST, MCP and jobs; migrations need to be
// present at first run without reading from disk.
//
//go:embed *.sql
var FS embed.FS