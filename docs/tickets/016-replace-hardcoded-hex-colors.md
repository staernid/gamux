# 016 — Replace hard-coded hex colors with theme tokens

**Area:** components · **Size:** M (can be split by folder)

## Context
There is a **light theme** (`style.css` defines `--surface-*` for both dark and
light, exposed as Tailwind `surface-*` colors in `tailwind.config.js`). About
120 hard-coded hex colors in `.svelte` files ignore it, so those spots look
wrong in light mode. Worst files:

```
schema/controls/EnvEditorControl.svelte   8
schema/controls/PathPickerControl.svelte  6
views/hub/HubCommunityTab.svelte          5
views/hub/cards/HubLogsCard.svelte        5
common/GhostInput.svelte                  4
blocks/GameHeroBackdrop.svelte            4
blocks/AchievementRibbon.svelte           4
```
Full list: `grep -rcE '#[0-9a-fA-F]{6}\b' --include=*.svelte frontend/src | grep -v ':0$'`

## Do
1. For each hex color, map it to the nearest `surface-*` token (backgrounds
   and borders) or a Tailwind neutral (text). Monochrome only; semantic color
   only for active/destructive states (AGENTS.md → UI Design).
2. Exceptions you may keep: colors over cover art or hero images that must
   stay dark in both themes (e.g. gradient scrims in `GameHeroBackdrop`). Add a
   short comment explaining why.
3. Check both themes visually if you can (`npm run dev` in `frontend/`, toggle
   the theme in Settings).

## Done when
At most ~15 justified hex colors remain, and `make check-fast` is green.
