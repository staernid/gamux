# gbe_fork_helper

`gbe_fork_helper` is a utility tool designed to streamline the management of your `gbe_fork` (Goldberg Steam Emulator) installation. It helps in updating your `gbe_fork` directory and applying necessary configurations to your Steam API files.

## Features

- **Native Extraction**: Uses native Go libraries for `.7z` and `.tar.bz2` extraction, no external dependencies like `7z`.
- **Concurrent DLC Fetching**: Fetches DLC names in parallel for significantly faster configuration.
- **Concurrent Repository Updates**: Downloads and extracts Linux and Windows releases in parallel.
- **Structured Logging**: Uses `slog` for clean, professional output.
- **Modern CLI**: Powered by `urfave/cli/v2` with a built-in TUI.
- **Dry-run Mode**: Preview changes before applying them.
- **Custom Configuration**: Support for custom paths via configuration files.

## Usage

```bash
# Apply GBE to a game
gbe_fork_helper apply linux 123456

# Apply GBE with dry-run
gbe_fork_helper apply --dry-run linux 123456

# Start interactive TUI
gbe_fork_helper tui

# Update GBE fork from GitHub
gbe_fork_helper update

# Specify custom configuration file
gbe_fork_helper --config my_config.json update

# Display version
gbe_fork_helper --version
```

## Roadmap

### User Interface

- [x] Build a TUI using `bubbletea`.
- [ ] Research and select a Go GUI library (like Wails or Fyne).
- [ ] Create a new application entry point for the GUI.
- [ ] Connect the UI buttons to the core logic.

### Enhancements

- [ ] Add support for more Steam API variants.
- [x] Configuration file support for custom paths.
- [x] Dry-run mode for `apply`.
