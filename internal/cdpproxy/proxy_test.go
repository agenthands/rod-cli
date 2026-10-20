package cdpproxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
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

// TestTrafficRingBufferModel: Traffic() equals the last min(cap, n) entries of
// a plain list of logged messages. Sends are logged even when the inner Send
// fails; recvs are logged only on success.
func TestTrafficRingBufferModel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capN := rapid.IntRange(1, 8).Draw(t, "cap")
		ws := &fakeWS{}
		p := New(ws, capN, 0)
		var model []CDPMessage
		ops := rapid.SliceOfN(rapid.IntRange(0, 2), 0, 30).Draw(t, "ops")
		for i, op := range ops {
			frame := []byte(`{"method":"E","params":{"i":` + strconv.Itoa(i) + `}}`)
			switch op {
			case 0: // send; inner may fail
				ws.sendErr = nil
				if rapid.Bool().Draw(t, "sendFails") {
					ws.sendErr = errors.New("send failed")
				}
				_ = p.Send(frame)
				model = append(model, CDPMessage{Direction: "send", Raw: frame})
			case 1: // recv ok
				ws.readErr, ws.frames = nil, [][]byte{frame}
				if _, err := p.Read(); err != nil {
					t.Fatalf("Read: %v", err)
				}
				model = append(model, CDPMessage{Direction: "recv", Raw: frame})
			case 2: // recv err
				ws.readErr = errors.New("read failed")
				if _, err := p.Read(); err == nil {
					t.Fatal("expected read error")
				}
			}
		}
		if len(model) > capN {
			model = model[len(model)-capN:]
		}
		got := p.Traffic()
		if len(got) != len(model) {
			t.Fatalf("Traffic len %d, want %d", len(got), len(model))
		}
		for i := range model {
			if got[i].Direction != model[i].Direction || !bytes.Equal(got[i].Raw, model[i].Raw) {
				t.Fatalf("entry %d = %s %s, want %s %s", i, got[i].Direction, got[i].Raw, model[i].Direction, model[i].Raw)
			}
		}
	})
}

func TestTrafficDefaultCapAndNoAliasing(t *testing.T) {
	ws := &fakeWS{}
	p := New(ws, 0, 0)
	buf := []byte(`{"method":"E"}`)
	for i := 0; i < 300; i++ {
		_ = p.Send(buf)
	}
	if n := len(p.Traffic()); n != 256 {
		t.Fatalf("cap<=0: Traffic len %d, want 256", n)
	}
	buf[2] = 'X' // rod reuses its buffers after the call returns
	ws.frames = [][]byte{buf}
	if _, err := p.Read(); err != nil {
		t.Fatal(err)
	}
	buf[2] = 'Y'
	tr := p.Traffic()
	if string(tr[0].Raw) != `{"method":"E"}` || string(tr[255].Raw) != `{"Xethod":"E"}` {
		t.Fatalf("Traffic aliases caller buffers: first %s, last %s", tr[0].Raw, tr[255].Raw)
	}
}
