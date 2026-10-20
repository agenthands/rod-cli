package types

import (
	"context"
	"testing"

	"github.com/agenthands/godoll/network"
)

// routeCtx returns a context whose interceptor already exists (no page needed),
// so AddRoute/RemoveRoute rebuild its rules in place.
func routeCtx() *Context {
	ctx := NewContext(context.Background(), Config{})
	ctx.interceptor = network.NewInterceptor(nil)
	return ctx
}

// servedBody is what godoll serves for url: the body of the first rule that matches.
func servedBody(ctx *Context, url string) string {
	for _, r := range ctx.interceptor.Rules() {
		if network.MatchURL(url, r.URLPattern) {
			return r.MockResponse.Body
		}
	}
	return ""
}

func TestRouteRemovePartialThenLastDropsInterceptor(t *testing.T) {
	ctx := routeCtx()
	ctx.AddRoute("**/*.png", "png")
	ctx.AddRoute("**/api/**", "api")

	ctx.RemoveRoute("**/*.png")
	if got := servedBody(ctx, "https://x.test/a.png"); got != "" {
		t.Fatalf("removed route still served %q", got)
	}
	if got := servedBody(ctx, "https://x.test/api/v1"); got != "api" {
		t.Fatalf("remaining route served %q, want api", got)
	}

	ctx.RemoveRoute("**/api/**")
	if ctx.interceptor != nil {
		t.Fatal("interceptor kept after the last route was removed")
	}
}

func TestRouteNewestOverlappingPatternWins(t *testing.T) {
	for i := range 50 {
		ctx := routeCtx()
		ctx.AddRoute("**/api/**", "broad")
		ctx.AddRoute("**/api/users", "narrow")
		ctx.AddRoute("**/api/users*", "newest")
		if got := servedBody(ctx, "https://x.test/api/users"); got != "newest" {
			t.Fatalf("run %d: served %q, want newest", i, got)
		}
		ctx.AddRoute("**/api/**", "broad-again") // re-adding makes it the newest
		if got := servedBody(ctx, "https://x.test/api/users"); got != "broad-again" {
			t.Fatalf("run %d: after re-add served %q, want broad-again", i, got)
		}
	}
}
