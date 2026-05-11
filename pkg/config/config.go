// Package config wraps Kong with conventions used across services.
// Use this so the env-prefix and help-flag behave the same everywhere.
package config

import (
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// Parse fills `cfg` from CLI flags and environment variables.
// Env vars are prefixed by `BASE_<SERVICE>_` (uppercase).
func Parse(serviceName string, cfg any, args ...string) {
	if args == nil {
		args = os.Args[1:]
	}
	prefix := "BASE_" + strings.ToUpper(strings.ReplaceAll(serviceName, "-", "_"))
	parser, err := kong.New(cfg,
		kong.Name(serviceName),
		kong.Description("base-microservice "+serviceName+" binary"),
		kong.DefaultEnvars(prefix),
		kong.UsageOnError(),
	)
	if err != nil {
		panic(err)
	}
	if _, err := parser.Parse(args); err != nil {
		parser.FatalIfErrorf(err)
	}
}
