# rod-cli Plugin Documentation

This is the index for writing, loading, and running `rod-cli` plugins — JavaScript files with optional lifecycle hooks that run in the goja sandbox alongside your browser session. New to plugins? Start with the [Writing Your First Plugin](./authoring.md) tutorial, then use the reference pages and worked examples below.

## Getting Started

- [Writing Your First Plugin](./authoring.md) — zero-to-running tutorial: what a plugin is, where the script lives, writing a hook, loading it, and inspecting results.

## Reference

- [Lifecycle Hooks](./lifecycle-hooks.md) — the four lifecycle hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) and their CDP payloads.
- [State / Context API](./state-api.md) — the `api` global accessors available inside hooks and getters.
- [Plugin CLI Reference](./cli-reference.md) — `plugin load`, `plugin list`, and `plugin run`, with arguments, output, and exit behavior.

## Examples

- [Starter Template](./examples/starter.md) — a copyable scaffold to begin from.
- [Per-Hook Recipes](./examples/recipes.md) — focused single-hook recipes you can lift into your own plugin.
- [XSS Scanner](./examples/xss-scanner.md) — the flagship worked example, end to end.

## See Also

- Top-level [README](../../README.md) — project overview and the full documentation index.
