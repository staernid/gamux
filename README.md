# gbe_fork_helper

`gbe_fork_helper` is a utility tool designed to streamline the management of your `gbe_fork` (Goldberg Steam Emulator) installation. It helps in updating your `gbe_fork` directory and applying necessary configurations to your Steam API files.

Usage:
```
Usage: gbe_fork_helper <command> [options]

Commands:
            apply <platform> <appid> - Apply GBE to Steam API files and configure DLCs
            update                   - Update the GBE fork repository
            version                  - Display the application version
```

## Roadmap

### Improve Platform Compatibility:

        [ ] Abstract external commands (7z) with native Go libraries for archive extraction

### User Interface

        [ ] Research and select a Go GUI library (like Fyne, Wails, or Gio).

        [ ] Create a new application entry point for the GUI (e.g., in cmd/gbe_gui/main.go).

        [ ] Build the UI components (buttons for "Update", "Apply", a list for DLCs, etc.).

        [ ] Connect the UI buttons to call the functions in your refactored gbe, updater, and steam packages. Should be a thin layer.
