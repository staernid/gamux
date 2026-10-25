# 017 — Remove @ts-ignore / {any} escape hatches

**Area:** components · **Size:** M (do a few files per session)

## Context
`svelte-check` type-checks the JSDoc-annotated JS. About 123 `@ts-ignore`,
`@ts-nocheck` or `{any}` occurrences hide real type information. Worst files:
`schema/PropertySheet.svelte` (10), `views/hub/cards/HubIntegrityCard.svelte`
(6), `blocks/PlayActionCluster.svelte` (6), `views/tasks/TaskCard.svelte` (5),
`views/hub/HubBetasTab.svelte`, `stage/catalog/CatalogCard.svelte`,
`schema/controls/EnvEditorControl.svelte`, `domain/MetadataLinker.svelte` (4
each).
List: `grep -rcE '@ts-ignore|@ts-nocheck|\{any\}' frontend/src | grep -v ':0$' | sort -t: -k2 -nr`

## Do
- Replace `{any}` with types from `frontend/src/types/domain.d.ts`
  (`import('../../types/domain').Game`, etc.). Add missing interfaces there.
- Remove each `@ts-ignore` by fixing the underlying type issue (usually a
  missing `$props()` JSDoc type, a nullable value, or an event target cast).
- Don't replace `@ts-ignore` with `@ts-expect-error` or casts to `any`.

## Done when
The files you touched have none left, `npm run check` reports 0 errors and 0
warnings, and `make check-fast` is green.
