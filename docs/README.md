# sgrbot Project Summary

## Project Overview

**sgrbot** is a Discord bot written in Go, designed with a modular architecture
that follows Domain-Driven Design (DDD) principles and Test-Driven Development
(TDD) practices.

## Design Philosophy

### Modular Architecture

sgrbot employs a modular architecture. The core functionality of the bot is
implemented in `internal/bot`, while additional features are implemented as
modules under `internal/modules`.

Modules register themselves via `init()` functions, allowing the main package
to include them with blank imports:

```go
import (
    _ "github.com/sglre6355/sgrbot/internal/modules/example"
)
```

### Domain-Driven Design (DDD)

Each module follows a layered architecture:

- **Domain Layer**: Pure business logic, entities, value objects, and
  repository interfaces. No external dependencies.
- **Application Layer**: Application logic (interactors). Depends only on
  domain and defines port interfaces for infrastructure.
- **Infrastructure Layer**: Implements interfaces defined by domain/application
  (adapters). Handles external systems (e.g. databases).
- **Presentation Layer**: Discord-facing code (commands, handlers). Translates
  between Discord API and application/usecase logic.

### Test-Driven Development (TDD)

- Domain and application/usecase layers have comprehensive unit tests
- Tests use mock implementations of interfaces
- Test files are co-located with implementation (`*_test.go`)
- Adheres to t_wada's TDD principles

## Tech Stack

| Component | Technology |
| --------- | ---------- |
| Language | Go |
| Discord Library | `github.com/bwmarrin/discordgo` |
| Config | `github.com/caarlos0/env/v11` (environment variables) |
| Logging | `log/slog` (structured JSON logging) |

## Project Structure

```text
sgrbot/
├── cmd/
│   └── sgrbot/
│       └── main.go              # Entry point, signal handling
├── internal/
│   ├── bot/
│   │   ├── bot.go               # Bot lifecycle, session management
│   │   ├── config.go            # Environment config loading
│   │   ├── module.go            # Module interface definition
│   │   ├── registry.go          # Global module registry
│   │   └── responder.go         # Discord response helpers
│   └── modules/
└── go.mod
```

## Core Interfaces

### `Module` Interface (`internal/bot/module.go`)

```go
type Module interface {
    Name() string
    Commands() []*discordgo.ApplicationCommand
    CommandHandlers() map[string]InteractionHandler
    EventHandlers() []EventHandler
    Init(deps ModuleDependencies) error
    Shutdown() error
}
```

### `ConfigurableModule` Interface (`internal/bot/module.go`)

Modules can optionally implement `ConfigurableModule` to load and validate
module-specific configuration before `Init()` and before the Discord connection
is established.

```go
type ConfigurableModule interface {
    LoadConfig() error
}
```

## Bot Lifecycle

1. `main()` loads config and creates `Bot`
2. `Bot.LoadModules()` retrieves modules from global registry
3. `Bot.Start()`:
   - Loads module configuration (optional `ConfigurableModule.LoadConfig()`)
   - Creates Discord session
   - Registers interaction handler
   - Opens Discord connection
   - Calls `module.Init()` for each module (after Open so session state is available)
   - Builds handler map
   - Registers event handlers
   - Registers slash commands
4. Wait for SIGINT/SIGTERM
5. `Bot.Stop()` calls `module.Shutdown()` and closes session

## Configuration

Environment variables:

- `DISCORD_TOKEN`: (required)

## Running the Bot

```sh
# Development
DISCORD_TOKEN=your_token go run ./cmd/sgrbot

# Build with version
go build -ldflags "-X main.version=1.0.0" ./cmd/sgrbot
```

## Testing

```sh
# Run all tests
go test -race ./...

# Run with coverage
go test -race -cover ./...

# Run specific module tests
go test -race ./internal/modules/example/...
```
