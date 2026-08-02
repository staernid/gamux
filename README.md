# gamux

Post-download Steam game manager for Linux (DRM removal via Goldberg Emulator, Lutris registration, Steam shortcuts).

> **Note**: Active prototyping stage. Architecture and CLI interfaces are subject to change.

## Quick Start

```bash
make build   # Build binary to ./gamux
make run     # Run binary directly
make test    # Run unit tests
make lint    # Run linter
```

## Basic Usage

```bash
./gamux apply linux <appid>
./gamux update
./gamux tui
```
