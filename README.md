# pen-bot

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Pen bot is a community-built Discord bot for the [Neon Genesis Linux](https://discord.gg/nglinux) community, written in Go. It leverages [Disgo](https://github.com/DisgoOrg/disgo) for Discord integration.

## Architecture

This repository follows a "monorepo-like" structure using Go modules, allowing modular bot construction. The codebase is divided into the following modules:

- **core**: Shared code and utilities used across all modules.
- **community**: Snippets and fun commands for community engagement.
- **moderation**: Moderation actions including jail, ban, mute, and other related functions.
- **admin**: Reserved for future administrative features (no planned features yet).

This modular design enables users to build custom bots by selecting and combining only the necessary modules. It supports multiple bots with different purposes and Discord permission requirements from a single codebase, facilitating easy deployment orchestration.

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

3. Install dependencies:
   ```bash
   go mod download
   ```

## Usage

### Configuration

Create a `.env` file in the root directory with your bot token:

```env
DISCORD_BOT_TOKEN=your_bot_token_here
```

**Note**: Bot tokens are sensitive information. Manage them securely at the [Discord Developer Portal](https://discord.com/developers/applications).

### Deployment

Deploy the bot using Docker Compose:

```bash
docker-compose up --build
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

