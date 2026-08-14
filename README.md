# gamux

Steam game manager and downloader for Linux.

## Project Goals

- **Direct Depot Downloads**: Download game depots and updates directly from Steam.
- **Automated DRM Emulation**: Streamline Goldberg Steam Emulator setup for downloaded games.
- **Ecosystem Integration**: Register games automatically with Lutris (YAML configuration & `pga.db` database integration).
- **File Safety**: Provide non-destructive setup with atomic backups, rollback support, and dry-run previews.
- **Zero-Friction UX**: Deliver an intuitive, self-guiding CLI experience requiring minimal manual configuration.

## Development & Releasing

> [!IMPORTANT]
> Steamless release binaries (`steamless/bin/*`) are not stored in the repository. Be sure to run `make update-steamless` before building or running tests.

- **Prerequisite**: Fetch the Steamless release binaries from [`steamless-rs`](https://github.com/staernid/steamless-rs):
  ```bash
  make update-steamless
  ```
- **Tooling**: Run `make test`, `make lint`, and `make build` to verify changes.
- **Tagging & Releasing**: Version sequentially (`v1.0.104` -> `v1.0.105` -> `v1.0.106`). Create and push tags (`git tag v1.0.104 && git push origin v1.0.104`) to trigger GitHub Actions automated builds.

