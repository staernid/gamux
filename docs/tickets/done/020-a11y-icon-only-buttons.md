# 020 — Accessibility: label icon-only buttons and check focus states

**Area:** components · **Size:** M

## Context
There are 255 `<button>`s and 68 `aria-label`s. Buttons with visible text
don't need one, but icon-only buttons (`<Icon .../>` as the only child) do.
Keyboard and gamepad navigation also depend on visible focus styles.

## Do
1. Find icon-only buttons: buttons whose only child is `<Icon` or an `<svg>`.
   A rough search: `grep -rn -A2 "<button" frontend/src --include=*.svelte | grep -B1 "<Icon"`.
2. Add a concise `aria-label` (and `title` if the UI has no tooltip).
3. Make sure every interactive element has a visible focus style
   (`focus-visible:ring-*` or similar, monochrome).
4. Fix the one `svelte-ignore a11y` suppression properly if possible.
5. Don't change the layout or visual design.

## Done when
Every icon-only button has an accessible name, and `make check-fast` is green.
