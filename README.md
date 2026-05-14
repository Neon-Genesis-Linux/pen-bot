# pen-bot

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Pen bot is a community-built Discord bot for the [Neon Genesis Linux](https://discord.gg/nglinux) community, written in Go. It leverages [Disgo](https://github.com/DisgoOrg/disgo) for Discord integration.

## Architecture

The repository structure follows the practices outlined in [golang-standards/project-layout](https://github.com/golang-standards/project-layout/blob/master/README.md).
It is a monorepo with a single root Go module and multiple bot binaries.

- `cmd/` contains `main.go` entrypoints for all the resulting bot binaries:
  - `pen-fun/`: the first bot, using the `community` package
- `internal/` contains reusable library packages:
  - `community/`: community engagement and fun commands
  - `core/`: shared bot startup and common utilities
  - `moderation/`: moderation actions

This design keeps shared code in one repository while letting each bot import only the packages it needs. Modules register their commands with the core registry for centralized dispatch.

## Installation

### Prerequisites

- Go (see [Learning Go](https://github.com/mauricioabreu/golings) for resources)
- Docker and Docker Compose (for deployment)
  - Podman and Podman Compose can also be used
- Nix (optional, for development environment)

### Development Setup

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd pen-bot
   ```

2. (Optional) Use Nix for a consistent development environment:
   ```bash
   nix develop
   ```

3. Install dependencies for the repository:
   ```bash
   go mod tidy
   ```

## Usage

### Configuration

Create a `.env` file in the root directory with your bot token:

```env
BOT_TOKEN=your_bot_token_here
```

**Note**: Bot tokens are sensitive information. Manage them securely at the [Discord Developer Portal](https://discord.com/developers/applications).

### Deployment

Deploy the pen-fun bot using Docker Compose:

```bash
docker-compose up
```

Or build the bot directly:

```bash
go build -o pen-fun ./cmd/pen-fun
```

The `docker-compose.yml` file handles the build and deployment process with hot-reload for development.

## Contributing

Contributions are welcome! Please follow the established code style and include tests for new features.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Resources

- [Disgo](https://github.com/DisgoOrg/disgo) - Discord library for Go
- [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests) - Test-driven Go learning
- [Golings](https://github.com/mauricioabreu/golings) - Learning Go exercises
- [MCP for Developers](https://github.com/mrjoshuak/godoc-mcp) - Go documentation tool for LLM agents
- [Uber Go Style Guide](https://github.com/uber-go/guide) - Go coding standards at Uber
- [Golang Standards project layout](https://github.com/golang-standards/project-layout)
