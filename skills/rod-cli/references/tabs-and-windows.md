# Tabs and Windows Management

By default, `rod-cli` operates on a single active browser tab. However, many workflows require opening new tabs, managing popups, or switching between contexts.

## Listing Tabs
To view all currently open tabs in the active browser session:
```bash
rod-cli tab-list
```
This will output a numbered list of tabs. The index numbers are used to target tabs in subsequent commands.

## Opening a New Tab
You can explicitly spawn a new tab and navigate it to a URL:
```bash
rod-cli tab-new https://example.com
```
*Note: This automatically switches the active context to the new tab.*

## Switching Active Tabs
If an action (like clicking a link with `target="_blank"`) opens a new tab, you must explicitly switch your CLI context to that new tab before you can interact with it.

1. First, list the tabs to find the index of your new tab:
```bash
rod-cli tab-list
```
2. Then, select the tab by its index:
```bash
rod-cli tab-select 1
```

## Closing Tabs
To clean up tabs you no longer need:
```bash
# Close the tab at index 1
rod-cli tab-close 1
```
