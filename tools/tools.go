//go:build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/v2/pkg/commands"
	_ "github.com/pressly/goose/v3"
	_ "github.com/swaggo/swag/v2/gen"
	_ "github.com/vektra/mockery/v2/pkg"
)
