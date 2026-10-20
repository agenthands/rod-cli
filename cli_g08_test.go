package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	"pgregory.net/rapid"
)

// TestReorderGlobalFlagsProperties (root-2): argv built from labelled segments
// (global bools, value flags in both forms, plain tokens, an optional `--` tail)
// is reordered into a permutation with [0] fixed, every global segment intact,
// the plain tokens in their order after the globals, the tail a byte-identical
// suffix, and a second pass changes nothing.
func TestReorderGlobalFlagsProperties(t *testing.T) {
	app := getApp()
	var bools, values []string
	takesValue := map[string]bool{}
	for _, f := range app.Flags {
		tv := f.(cli.DocGenerationFlag).TakesValue()
		for _, n := range f.Names() {
			takesValue[n] = tv
			if tv {
				values = append(values, n)
			} else {
				bools = append(bools, n)
			}
		}
	}
	dashes := rapid.SampledFrom([]string{"-", "--"})
	notDash := rapid.String().Filter(func(s string) bool { return !strings.HasPrefix(s, "-") })
	plain := rapid.OneOf(
		rapid.SampledFrom([]string{"-", "---", "--out", "-10", "goto", "https://x"}),
		rapid.StringMatching(`-[0-9]{1,4}`),
		notDash,
	)

	rapid.Check(t, func(t *rapid.T) {
		arg0 := notDash.Draw(t, "arg0")
		var globals [][]string // each global segment, in input order
		var plains []string
		in := []string{arg0}
		for i, n := 0, rapid.IntRange(0, 12).Draw(t, "n"); i < n; i++ {
			switch rapid.IntRange(0, 3).Draw(t, "kind") {
			case 0:
				seg := []string{dashes.Draw(t, "d") + rapid.SampledFrom(bools).Draw(t, "bool")}
				globals = append(globals, seg)
				in = append(in, seg...)
			case 1:
				seg := []string{dashes.Draw(t, "d") + rapid.SampledFrom(values).Draw(t, "val") + "=" + rapid.String().Draw(t, "v")}
				globals = append(globals, seg)
				in = append(in, seg...)
			case 2:
				v := rapid.String().Filter(func(s string) bool { return s != "--" }).Draw(t, "v")
				seg := []string{dashes.Draw(t, "d") + rapid.SampledFrom(values).Draw(t, "val"), v}
				globals = append(globals, seg)
				in = append(in, seg...)
			default:
				p := plain.Draw(t, "plain")
				plains = append(plains, p)
				in = append(in, p)
			}
		}
		var tail []string
		if rapid.Bool().Draw(t, "dd") {
			tok := rapid.OneOf(rapid.String(), plain, rapid.Map(rapid.SampledFrom(bools), func(n string) string { return "--" + n }),
				rapid.Map(rapid.SampledFrom(values), func(n string) string { return "-" + n }))
			tail = append([]string{"--"}, rapid.SliceOf(tok).Draw(t, "tail")...)
			in = append(in, tail...)
		}

		out, err := reorderGlobalFlags(app, append([]string(nil), in...))
		if err != nil {
			t.Fatalf("reorderGlobalFlags(%q): %v", in, err)
		}

		if len(out) != len(in) || out[0] != in[0] || !sameMultiset(out, in) {
			t.Fatalf("not a permutation with [0] fixed: in %q out %q", in, out)
		}
		k := 0
		for _, g := range globals {
			k += len(g)
		}
		wantRest := append(append([]string{}, plains...), tail...)
		if got := out[1+k:]; strings.Join(got, "\x00") != strings.Join(wantRest, "\x00") || len(got) != len(wantRest) {
			t.Fatalf("plain tokens + tail after globals: got %q, want %q (in %q)", got, wantRest, in)
		}
		// Re-segment the global prefix: every value flag without '=' must be
		// followed directly by its value, so the segments match the input's.
		var gotSegs []string
		for j := 1; j < 1+k; j++ {
			name, hasValue := globalFlagName(out[j])
			tv, known := takesValue[name]
			if name == "" || !known {
				t.Fatalf("non-global %q in the global prefix of %q", out[j], out)
			}
			seg := out[j]
			if tv && !hasValue {
				j++
				seg += "\x00" + out[j]
			}
			gotSegs = append(gotSegs, seg)
		}
		var wantSegs []string
		for _, g := range globals {
			wantSegs = append(wantSegs, strings.Join(g, "\x00"))
		}
		if !sameMultiset(gotSegs, wantSegs) {
			t.Fatalf("global segments: got %q, want %q", gotSegs, wantSegs)
		}
		if again, _ := reorderGlobalFlags(app, append([]string(nil), out...)); strings.Join(again, "\x00") != strings.Join(out, "\x00") {
			t.Fatalf("not idempotent: %q -> %q", out, again)
		}
	})
}

func sameMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

// TestTrailingGlobalFlagWithoutValueErrors (root-3): a value-taking global flag
// at the end of argv has no value; it must not swallow the subcommand.
func TestTrailingGlobalFlagWithoutValueErrors(t *testing.T) {
	clearPortFiles(t)
	oldArg0 := os.Args[0]
	os.Args[0] = filepath.Join(os.TempDir(), "rod-cli-definitely-missing-binary-xyz")
	t.Cleanup(func() { os.Args[0] = oldArg0 })

	for _, args := range [][]string{{"sessions", "--session"}, {"goto", "https://x", "-s"}} {
		var err error
		captureStdout(t, func() { err = runApp(args...) })
		// The spawn path must not be reached: the flag error comes first.
		if err == nil || strings.Contains(err.Error(), "ensure daemon") {
			t.Errorf("runApp(%q) = %v, want an argv error", args, err)
		}
	}
}
