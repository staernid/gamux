# 004 — Normalize tool status entities

**Area:** `components/views/settings/ToolsMaintenanceTab.svelte` · **Size:** S

## Context
ToolsMaintenanceTab guesses at field names:
```js
if (typeof tool.is_installed === 'boolean') return tool.is_installed;   // ~line 41
{tool.name || tool.display_name || tool.key}                            // ~line 165
updateTool(tool.key || tool.tool_name, tool.name || tool.display_name)  // ~line 207
```
The data comes from `AppAPI.getToolsStatus()` → Go `App.GetToolsStatus` (in
`gui/`). Follow it to the Go struct and read the real JSON tags.

## Do
1. List the real Go JSON fields for a tool entry.
2. Add `normalizeTool(raw)` (put it in `utils/` or a small `stores/tools`
   helper) that outputs camelCase with exactly one source per field.
3. Use it at the fetch site and remove every `||` alias chain in the component.
4. Update `MOCK_TOOLS_STATUS` in `api/mockData.js` to the real Go shape.

## Done when
There are no alias chains for tool fields, and `make check-fast` is green.
