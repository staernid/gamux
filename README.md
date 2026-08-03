# gamux

Post-download Steam game manager for Linux.

## Project Goals

- **Automated DRM Emulation**: Streamline Goldberg Steam Emulator setup for downloaded games.
- **Ecosystem Integration**: Register games automatically with Lutris and Steam (shortcuts & grid artwork).
- **File Safety**: Provide non-destructive setup with atomic backups, rollback support, and dry-run previews.
- **Zero-Friction UX**: Deliver an intuitive, self-guiding CLI experience requiring minimal manual configuration.

## Development & Releasing

- **Tooling**: Run `make test`, `make lint`, and `make build` to verify changes.
- **Tagging & Releasing**: Push a semver tag (`git tag vX.Y.Z && git push origin vX.Y.Z`) to trigger the GitHub Actions build and release workflow.
