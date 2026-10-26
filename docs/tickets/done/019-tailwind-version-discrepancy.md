# 019 — Tailwind v3 vs v4 discrepancy (NEEDS HUMAN DECISION)

**Area:** build / docs · **Size:** S (doc fix) or L (migration)

## Context
`AGENTS.md` says the GUI uses "Tailwind v4", but `frontend/package.json` has
`tailwindcss ^3.4.17` with a v3 JS config (`tailwind.config.js`,
`postcss.config.js`, `autoprefixer`).

## Options
- **A (cheap):** change AGENTS.md to say Tailwind v3. Done.
- **B (migration):** move to v4 with `@tailwindcss/vite`. The config becomes
  CSS-first: the `surface-*` colors and the `2xs/3xs/4xs` font sizes move into
  an `@theme` block in `style.css`, and PostCSS/autoprefixer are removed. v4
  renames some utilities (e.g. `shadow-sm`→`shadow-xs`, `ring` default width,
  `bg-opacity-*` removed). Run the official `npx @tailwindcss/upgrade` and then
  review every changed class visually in both themes.

## Do
Ask the maintainer which option they want. Don't pick B on your own.
