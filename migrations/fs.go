// Package migrations holds golang-migrate SQL pairs. The CLI applies them via
// make migrate-up; integration tests embed SQL and apply it to an isolated container.
package migrations

import "embed"

// SQL is the golang-migrate pair set. Integration tests apply it to an isolated container.
//
//go:embed *.sql
var SQL embed.FS
