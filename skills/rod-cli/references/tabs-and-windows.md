# Tabs and Windows Management

By default, `rod-cli` operates on a single active browser tab. However, many workflows require opening new tabs, managing popups, or switching between contexts.

## Listing Tabs
To view all currently open tabs in the active browser session:
```bash
rod-cli tab-list
```
Each line is `[index] <ID> <Title> (<url>)`; the current tab is marked `*`:
```
* [0] 3F2A9C1B Example Domain (https://example.com/)
  [1] 9D04E7AA Docs (https://example.com/docs)
```
Target a tab by its index or by its 8-character ID (any unique, case-insensitive ID prefix works; fewer than 8 digits is read as an index). Prefer the ID: indexes shift when tabs open or close, IDs do not.

**Ownership rule:** Drive only tabs you created (IDs printed by `tab-new`); under `--cdp-endpoint` never navigate the user's existing tabs.

## Opening a New Tab
You can explicitly spawn a new tab and navigate it to a URL:
```bash
rod-cli tab-new https://example.com
```
*Note: This automatically switches the active context to the new tab. Switching tabs clears the snapshot, so take a new `snapshot` before using refs. Mock routes follow the current tab.*

## Switching Active Tabs
If an action (like clicking a link with `target="_blank"`) opens a new tab, you must explicitly switch your CLI context to that new tab before you can interact with it.

1. First, list the tabs to find the ID of your new tab:
```bash
rod-cli tab-list
```
2. Then, select the tab by its ID (or index):
```bash
rod-cli tab-select 9D04E7AA
```
A tab rod-cli did not open itself (a popup or `target="_blank"` link) is prepared on first select and the output says `adopted, stealth applies from next navigation (reload to apply)`: its already-loaded document runs without stealth evasions until you `reload` or navigate.

## Closing Tabs
To clean up tabs you no longer need:
```bash
# Close a tab by ID (or index)
rod-cli tab-close 9D04E7AA
```
Closing the current tab switches to the tab before it and reports `; current: [j] <ID> ...`. Closing the last tab leaves no current tab; the next `goto` opens one.
