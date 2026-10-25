# 021 — Move DEFAULT_MOCKS out of api/app.js and match the Go wire shape

**Area:** `frontend/src/api/` · **Size:** S–M

## Context
`api/app.js` is mostly a ~300-line `DEFAULT_MOCKS` object (browser preview and
Vitest backend) with a small bridge at the bottom. Some mocks return shapes the
real backend never sends (e.g. `linkGameMetadata` returns `app_id` **and**
`AppID`; `inspectGame` matches `g.game_dir || g.GameDir`). Wrong mocks hide
normalization bugs, because tests pass against data the real backend never
produces.

`api/app.test.js` already guarantees every mock key is a real bound method.

## Do
1. Move `DEFAULT_MOCKS` to `api/mocks.js` (alongside `mockData.js`). `app.js`
   imports it. Keep the `DEFAULT_MOCKS` export from `app.js` (re-export) or
   update `app.test.js`.
2. Make each mock return exactly the Go JSON shape: snake_case per the Go
   struct tags, no PascalCase or camelCase duplicates. Look up the return type
   in the comment next to each method in `api/bindings.gen.js`, then read that
   Go struct's tags.
3. Fix any tests that break because they relied on the duplicate keys.

## Done when
`app.js` is under ~150 lines, no mock returns duplicate-cased keys, and
`make check-fast` is green.
