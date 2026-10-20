package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	"pgregory.net/rapid"
)

// fwdSentinel is the --proxy-auth credential every draw sets. It must reach
// the daemon through its environment only.
const fwdSentinel = "u53r:p4ss-G09-sentinel"

// fwdExcluded are the global flags the client never forwards: output knobs,
// the session (always forwarded first) and the out-of-band credential.
var fwdExcluded = map[string]bool{
	"banner": true, "raw": true, "json": true, "max-output": true, "session": true, "proxy-auth": true,
}

// fwdDraw is one generated client invocation: the value of each flag it sets.
type fwdDraw struct {
	args []string
	set  map[string]any
}

// drawClientFlags sets each forwardable flag of getApp() or leaves it unset,
// by name or alias, in --name=value form.
func drawClientFlags(t *rapid.T) fwdDraw {
	d := fwdDraw{set: map[string]any{}}
	for _, f := range getApp().Flags {
		name := f.Names()[0]
		if fwdExcluded[name] || !rapid.Bool().Draw(t, "set-"+name) {
			continue
		}
		var v any
		var s string
		switch f.(type) {
		case *cli.StringFlag:
			str := rapid.StringMatching(`[^\x00]+`).Filter(func(s string) bool {
				return !(name == "profile" && s == profileListValue)
			}).Draw(t, name)
			v, s = str, str
		case *cli.BoolFlag:
			b := rapid.Bool().Draw(t, name)
			v, s = b, strconv.FormatBool(b)
		case *cli.IntFlag:
			n := rapid.Int().Draw(t, name)
			v, s = n, strconv.Itoa(n)
		case *cli.Float64Flag:
			x := rapid.Float64().Filter(func(x float64) bool { return !math.IsInf(x, 0) && !math.IsNaN(x) }).Draw(t, name)
			v, s = x, strconv.FormatFloat(x, 'g', -1, 64)
		default:
			t.Fatalf("flag %s has unhandled type %T", name, f)
		}
		d.set[name] = v
		d.args = append(d.args, "--"+rapid.SampledFrom(f.Names()).Draw(t, "alias-"+name)+"="+s)
	}
	d.args = append(d.args, "--proxy-auth="+fwdSentinel)
	return d
}

// fwdRig swaps os.Args[0] for a /bin/sh recorder that writes the daemon argv
// and env NUL-separated under dir/<session>.{args,env} and exits 1.
type fwdRig struct{ tmp, dir string }

func newFwdRig(t *testing.T) fwdRig {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	if old, ok := os.LookupEnv("ROD_CLI_PROXY_AUTH"); ok {
		os.Unsetenv("ROD_CLI_PROXY_AUTH")
		t.Cleanup(func() { os.Setenv("ROD_CLI_PROXY_AUTH", old) })
	}
	dir := filepath.Join(tmp, "rec")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	rec := filepath.Join(tmp, "recorder.sh")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\0' \"$@\" > '%s'/\"$2\".args\nenv -0 > '%s'/\"$2\".env\nexit 1\n", dir, dir)
	if err := os.WriteFile(rec, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	old := os.Args[0]
	os.Args[0] = rec
	t.Cleanup(func() { os.Args[0] = old })
	return fwdRig{tmp: tmp, dir: dir}
}

// spawn runs the client with the drawn flags and returns the daemon argv and
// env the recorder saw.
func (r fwdRig) spawn(t *rapid.T, session string, d fwdDraw) (argv, env []string) {
	err := runApp(append(d.args, "--session", session, "snapshot")...)
	logPath := filepath.Join(r.tmp, "rod-cli-daemon.log")
	if err == nil || !strings.Contains(err.Error(), logPath) {
		t.Fatalf("spawn error %v does not name %s", err, logPath)
	}
	read := func(ext string) []string {
		b, err := os.ReadFile(filepath.Join(r.dir, session+ext))
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(string(bytes.TrimSuffix(b, []byte{0})), "\x00")
	}
	return read(".args"), read(".env")
}

// reparse runs the daemon argv through getApp() as main() would, with the
// hidden daemon command's Action replaced by capture.
func reparse(t *rapid.T, argv []string, capture func(c *cli.Context)) {
	app := getApp()
	for _, c := range app.Commands {
		if c.Name == "daemon" {
			c.Action = func(c *cli.Context) error { capture(c); return nil }
		}
	}
	full, err := reorderGlobalFlags(app, append([]string{"rod-cli"}, argv...))
	if err == nil {
		err = app.Run(full)
	}
	if err != nil {
		t.Fatalf("daemon argv %q does not parse: %v", argv, err)
	}
}

// TestFlagForwardingRoundTrip (root-1, daemon:D1): every flag the client sets
// survives the spawn argv with an equal value, gated flags stay unset when the
// client left them unset, and the credential travels only in the env.
func TestFlagForwardingRoundTrip(t *testing.T) {
	r := newFwdRig(t)
	i := 0
	rapid.Check(t, func(rt *rapid.T) {
		i++
		session := fmt.Sprintf("fwd-%d", i)
		d := drawClientFlags(rt)
		argv, env := r.spawn(rt, session, d)

		for _, a := range argv {
			if strings.Contains(a, fwdSentinel) {
				rt.Fatalf("credential in daemon argv: %q", argv)
			}
		}
		n := 0
		for _, e := range env {
			if strings.HasPrefix(e, "ROD_CLI_PROXY_AUTH=") {
				n++
				if e != "ROD_CLI_PROXY_AUTH="+fwdSentinel {
					rt.Fatalf("env carries %q", e)
				}
			}
		}
		if n != 1 {
			rt.Fatalf("ROD_CLI_PROXY_AUTH= appears %d times in env", n)
		}

		reparse(rt, argv, func(c *cli.Context) {
			if got := c.String("session"); got != session {
				rt.Fatalf("session %q, want %q", got, session)
			}
			for _, f := range c.App.Flags {
				name := f.Names()[0]
				if fwdExcluded[name] {
					continue
				}
				want, set := d.set[name]
				var got any
				switch ff := f.(type) {
				case *cli.StringFlag:
					got = c.String(name)
					if !set {
						want = ff.Value
					}
				case *cli.BoolFlag:
					got = c.Bool(name)
					if !set {
						want = ff.Value
					}
				case *cli.IntFlag:
					got = c.Int(name)
					if !set {
						want = ff.Value
					}
				case *cli.Float64Flag:
					got = c.Float64(name)
					if !set {
						want = ff.Value
					}
				}
				if got != want {
					rt.Fatalf("--%s: daemon read %#v, client set %#v (set=%v)", name, got, want, set)
				}
				// --headless/--vision forward only when true; compare by value.
				if name != "headless" && name != "vision" && c.IsSet(name) != set {
					rt.Fatalf("--%s: daemon IsSet=%v, client set=%v", name, c.IsSet(name), set)
				}
			}
		})
	})
}
