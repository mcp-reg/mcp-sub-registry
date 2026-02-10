# Contributing

Thanks for your interest in contributing to MCP Sub-Registry!

## Reporting Issues

Open an issue at [github.com/mcp-reg/mcp-sub-registry/issues](https://github.com/mcp-reg/mcp-sub-registry/issues) with:
- Steps to reproduce
- Expected vs actual behavior
- Go/Node versions

## Development Setup

**Prerequisites:** Go 1.23+, Node 20+

```bash
git clone https://github.com/mcp-reg/mcp-sub-registry.git
cd mcp-sub-registry
go mod download
cd web && npm install && cd ..
```

## Getting Started
```bash

# Run locally
make run

# Run tests
make test

# Build binary
make build
```

## Common Commands

```bash
make dev          # Run backend + frontend in dev mode
make test         # Run all tests
make lint         # Run linters
make build        # Build Go binary
make web-build    # Build frontend
make build-embed  # Build binary with embedded frontend
```

## Submitting PRs

1. Fork the repo and create a feature branch
2. Make your changes
3. Ensure `make test` and `make lint` pass
4. Submit a PR against `main`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
