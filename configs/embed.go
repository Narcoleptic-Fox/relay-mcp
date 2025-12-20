package configs

import "embed"

// Models contains embedded model configuration files
//
//go:embed models/*.json
var Models embed.FS

// CLIClients contains embedded CLI client configuration files
//
//go:embed cli_clients/*.json
var CLIClients embed.FS
