# 018 — Add tests for large untested components

**Area:** components · **Size:** M (one or two components per session)

## Context
These components hold significant logic but have no `*.test.js`:

| Component | Lines |
|---|---|
| `views/hub/HubConfigTab.svelte` | ~500 |
| `views/settings/cards/CoopLayoutCard.svelte` | ~415 |
| `views/settings/cards/CoopSeatCard.svelte` | ~300 |
| `views/hub/HubOverviewTab.svelte` | ~265 |
| `views/settings/cards/CoopStreamingCard.svelte` | ~230 |
| `schema/controls/PathPickerControl.svelte` | ~210 |
| `schema/controls/EnvEditorControl.svelte` | ~205 |

## Do
- Follow existing tests for style, e.g. `views/settings/CoopTab.test.js` and
  `common/StepperRow.test.js` (`@testing-library/svelte`, `vitest`).
- `AppAPI` calls are served by `DEFAULT_MOCKS` in `api/app.js`. Override them
  per test with `vi.spyOn(AppAPI, 'method').mockResolvedValue(...)`. Only spy
  on methods that exist in `api/bindings.gen.js` (others are `undefined`).
- Test behaviour (what the user sees and clicks), not internals. Aim for
  3–6 meaningful cases per component.
- If a component is too tangled to test, extract its pure logic into `utils/`
  and test that instead (and note it for ticket 014).

## Done when
Each component you picked has a test file, and `make check-fast` is green.
