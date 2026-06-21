# Phase 16: Add comprehensive test cases for godoll migration features - Plan

## Architecture & Approach

We will augment the existing test suite with targeted unit and integration tests that explicitly validate the edge cases introduced by the `godoll` migration:

1. **Transient Network Failure Resilience:** We will test that `godoll/retry` effectively catches failing navigations and DOM queries in `actions.go`, retrying them under simulated failure conditions until they succeed (or exhaust retries).
2. **Network Interceptor Logic:** We will write a unit test for `types.Context` to ensure `updateInterceptorRules()` correctly parses user-defined routes and injects the stealth profile headers into the `godoll/network.Interceptor`.
3. **Remote Browser Instantiation:** We will ensure `types.NewContext` falls back correctly or errors out gracefully when remote browser endpoints are malformed.

## Step-by-Step Implementation

1. **[actions_retry_test.go]**: Create a new test file in the `actions/` package that defines mock pages/elements simulating transient errors, and verify that `actions.Navigate` and `actions.Click` retry the configured number of times.
2. **[context_interceptor_test.go]**: Create a new test file in the `types/` package to instantiate a `Context` with mock routes, and verify that the underlying `godoll` interceptor rules match the routes perfectly.
3. **[Run Tests]**: Execute `go test ./...` and ensure coverage across the new edge cases.
