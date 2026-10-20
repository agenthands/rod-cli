package cdpproxy

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// fakeWS is an in-memory cdp.WebSocketable: Read pops queued frames, Send
// records frames and returns sendErr.
type fakeWS struct {
	frames  [][]byte
	readErr error
	sendErr error
	sent    [][]byte
}

func (f *fakeWS) Send(b []byte) error { f.sent = append(f.sent, b); return f.sendErr }

func (f *fakeWS) Read() ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	b := f.frames[0]
	f.frames = f.frames[1:]
	return b, nil
}

func readThrough(t *rapid.T, in []byte) []byte {
	out, err := New(&fakeWS{frames: [][]byte{in}}, 0, 0).Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return out
}

func toAny(t *rapid.T, b []byte) any {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("not JSON: %v: %s", err, b)
	}
	return v
}

var strGen = rapid.StringMatching(`[a-zA-Z_$<&é]{0,6}`)

func remoteObj(t *rapid.T, label string) map[string]any {
	return map[string]any{"type": "string", "value": strGen.Draw(t, label)}
}

// TestReadNormalizesGetPropertiesProperty: for Runtime.getProperties responses,
// Read output decodes to the input model with `value` removed only from
// descriptors that carry `get`; every other key survives.
func TestReadNormalizesGetPropertiesProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 5).Draw(t, "nprops")
		in, want := []any{}, []any{}
		hasGet := false
		for i := 0; i < n; i++ {
			d := map[string]any{"name": strGen.Draw(t, "name")}
			if rapid.Bool().Draw(t, "hasValue") {
				d["value"] = remoteObj(t, "value")
			}
			if rapid.Bool().Draw(t, "hasSet") {
				d["set"] = map[string]any{"type": "function"}
			}
			if rapid.Bool().Draw(t, "hasExtra") {
				d["configurable"] = rapid.Bool().Draw(t, "configurable")
				d["isOwn"] = rapid.Bool().Draw(t, "isOwn")
			}
			w := map[string]any{}
			for k, v := range d {
				w[k] = v
			}
			if rapid.Bool().Draw(t, "hasGet") {
				d["get"] = map[string]any{"type": "function", "className": "Function"}
				w["get"] = d["get"]
				delete(w, "value")
				hasGet = true
			}
			in, want = append(in, d), append(want, w)
		}
		id := rapid.IntRange(1, 1e6).Draw(t, "id")
		inMsg := map[string]any{"id": id, "result": map[string]any{"result": in}}
		wantMsg := map[string]any{"id": id, "result": map[string]any{"result": want}}
		extras := map[string]any{}
		if rapid.Bool().Draw(t, "internal") {
			extras["internalProperties"] = []any{map[string]any{"name": "[[Prototype]]", "value": remoteObj(t, "iv")}}
		}
		if rapid.Bool().Draw(t, "private") {
			extras["privateProperties"] = []any{map[string]any{"name": "#p", "value": remoteObj(t, "pv")}}
		}
		if rapid.Bool().Draw(t, "exception") {
			extras["exceptionDetails"] = map[string]any{"exceptionId": 1, "text": strGen.Draw(t, "exc")}
		}
		for k, v := range extras {
			inMsg["result"].(map[string]any)[k] = v
			wantMsg["result"].(map[string]any)[k] = v
		}
		if rapid.Bool().Draw(t, "session") {
			s := strGen.Draw(t, "sessionId")
			inMsg["sessionId"], wantMsg["sessionId"] = s, s
		}
		method := rapid.Bool().Draw(t, "method")
		if method {
			inMsg["method"] = "Runtime.getProperties"
		}

		input, _ := json.Marshal(inMsg)
		out := readThrough(t, input)

		if method || !hasGet {
			if !bytes.Equal(out, input) {
				t.Fatalf("expected byte-identical\n in: %s\nout: %s", input, out)
			}
		} else {
			wantJSON, _ := json.Marshal(wantMsg)
			if got, exp := toAny(t, out), toAny(t, wantJSON); !reflect.DeepEqual(got, exp) {
				t.Fatalf("normalized mismatch\n in: %s\ngot: %s\nwant: %s", input, out, wantJSON)
			}
		}
		if again := readThrough(t, out); !bytes.Equal(again, out) {
			t.Fatalf("Read∘Read not idempotent\nonce:  %s\ntwice: %s", out, again)
		}
	})
}
