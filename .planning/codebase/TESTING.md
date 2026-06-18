# Testing

**Mapped:** 2026-06-18

## Testing Strategy
- Currently, there is a lack of standard Go `*_test.go` files visible in the core directories (`/`, `tools/`, `types/`).
- Most testing appears to be manual or handled via integration scripts, given the nature of a CLI wrapper around an established automation library (`go-rod`).

## Framework
- Go's built-in `testing` package is expected to be the standard if unit tests are added.

## Execution
- No established test suite or CI verification logic is apparent from the top-level directory. Test coverage requires improvement to ensure stability for LLM agents.
