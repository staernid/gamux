# Developer Guide (`AGENTS.md`)

Essential guidelines for development on `gamux`.

## Core Conventions

1. **Logging Strategy**:
   - Use `log/slog` structured logging (`slog.Info`, `slog.Warn`, `slog.Error`).
   - Do not use `fmt.Println` or standard `log` package in library modules.
   - Note: In TUI mode, `slog` output is redirected to `gamux.log` or discarded.

2. **Error Handling**:
   - Wrap errors with context: `fmt.Errorf("...: %w", err)`.

3. **Concurrency & Rate Limits**:
   - Use `golang.org/x/sync/errgroup` with worker caps (e.g. `g.SetLimit(10)`) when querying external APIs.
   - Pass `context.Context` through all async/network chains.

4. **File Safety**:
   - Always back up libraries before patching (`util.BackupAndReplace`).
   - Respect `--dry-run` flags; log mutations with `[DRY RUN]` without modifying the filesystem.

5. **Testing**:
   - Mock HTTP requests using custom `http.RoundTripper`.
   - Clean up temporary files with `defer os.RemoveAll(tmpDir)`.
