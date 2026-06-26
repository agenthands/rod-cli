package cdpproxy

import (
	"encoding/json"
	"testing"
)

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// TestNormalizeGetProperties_StripsAccessorValue verifies that properties
// with a `get` field (accessor/descriptors) have their `value` stripped.
func TestNormalizeGetProperties_StripsAccessorValue(t *testing.T) {
	input := mustJSON(map[string]any{
		"id": 5,
		"result": map[string]any{
			"result": []map[string]any{
				{
					"name":         "stack",
					"value":        map[string]any{"type": "string", "value": "Error\n    at foo"},
					"get":          map[string]any{"type": "function", "className": "Function"},
					"writable":     true,
					"configurable": true,
					"enumerable":   false,
				},
			},
		},
	})

	output := normalizeCDPResponse(input)

	var out map[string]json.RawMessage
	if err := json.Unmarshal(output, &out); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}

	// Parse the normalized result
	var outer struct {
		Result []map[string]json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(out["result"], &outer); err != nil {
		t.Fatalf("result not valid: %v", err)
	}
	if len(outer.Result) != 1 {
		t.Fatalf("expected 1 property, got %d", len(outer.Result))
	}

	prop := outer.Result[0]
	if _, hasValue := prop["value"]; hasValue {
		t.Error("value should have been stripped from accessor property")
	}
	if _, hasGet := prop["get"]; !hasGet {
		t.Error("get should be preserved")
	}
	if string(prop["name"]) != `"stack"` {
		t.Errorf("name should be preserved, got %s", prop["name"])
	}
}

// TestNormalizeGetProperties_PreservesDataProperty verifies that non-accessor
// properties (no `get` field) keep their `value`.
func TestNormalizeGetProperties_PreservesDataProperty(t *testing.T) {
	input := mustJSON(map[string]any{
		"id": 5,
		"result": map[string]any{
			"result": []map[string]any{
				{
					"name":         "message",
					"value":        map[string]any{"type": "string", "value": "hello"},
					"writable":     true,
					"configurable": true,
					"enumerable":   true,
				},
			},
		},
	})

	output := normalizeCDPResponse(input)

	// Should be unchanged because there's no accessor property
	if string(output) != string(input) {
		t.Errorf("data property should pass through unchanged\ngot:  %s\nwant: %s", output, input)
	}
}

// TestNormalizeGetProperties_PassThroughNonGetProperties verifies non-Runtime
// messages pass through unchanged.
func TestNormalizeGetProperties_PassThroughNonGetProperties(t *testing.T) {
	tests := []struct {
		name  string
		input json.RawMessage
	}{
		{
			name:  "request",
			input: mustJSON(map[string]any{"id": 1, "method": "Page.navigate", "params": map[string]any{"url": "about:blank"}}),
		},
		{
			name:  "event",
			input: mustJSON(map[string]any{"method": "Page.frameNavigated", "params": map[string]any{"frame": map[string]any{"id": "abc"}}}),
		},
		{
			name:  "non-getProperties response",
			input: mustJSON(map[string]any{"id": 2, "result": map[string]any{"frameId": "abc"}}),
		},
		{
			name:  "error response",
			input: mustJSON(map[string]any{"id": 3, "error": map[string]any{"code": -32601, "message": "Method not found"}}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := normalizeCDPResponse(tt.input)
			if string(output) != string(tt.input) {
				t.Errorf("should pass through unchanged\ngot:  %s\nwant: %s", output, tt.input)
			}
		})
	}
}

// TestNormalizeGetProperties_InvalidJSON verifies garbage input returns unchanged.
func TestNormalizeGetProperties_InvalidJSON(t *testing.T) {
	tests := [][]byte{
		[]byte("not json"),
		[]byte(""),
		[]byte("{"),
		[]byte("null"),
		[]byte("42"),
	}

	for i, input := range tests {
		output := normalizeCDPResponse(input)
		if string(output) != string(input) {
			t.Errorf("test %d: garbage input should pass through unchanged", i)
		}
	}
}

// TestNormalizeGetProperties_MultipleProperties verifies mixed accessor+data
// properties are handled correctly.
func TestNormalizeGetProperties_MultipleProperties(t *testing.T) {
	input := mustJSON(map[string]any{
		"id": 5,
		"result": map[string]any{
			"result": []map[string]any{
				{
					"name":  "message",
					"value": map[string]any{"type": "string", "value": "hello"},
				},
				{
					"name":  "stack",
					"value": map[string]any{"type": "string", "value": "trace"},
					"get":   map[string]any{"type": "function"},
				},
				{
					"name":  "line",
					"value": map[string]any{"type": "number", "value": 42},
				},
			},
		},
	})

	output := normalizeCDPResponse(input)

	var out map[string]json.RawMessage
	json.Unmarshal(output, &out)

	var outer struct {
		Result []map[string]json.RawMessage `json:"result"`
	}
	json.Unmarshal(out["result"], &outer)

	if len(outer.Result) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(outer.Result))
	}

	// message: data property, value preserved
	if _, hasValue := outer.Result[0]["value"]; !hasValue {
		t.Error("message.value should be preserved (data property)")
	}
	// stack: accessor, value stripped
	if _, hasValue := outer.Result[1]["value"]; hasValue {
		t.Error("stack.value should be stripped (accessor property)")
	}
	// line: data property, value preserved
	if _, hasValue := outer.Result[2]["value"]; !hasValue {
		t.Error("line.value should be preserved (data property)")
	}
}

// TestNormalizeGetProperties_EmptyResult verifies an empty getProperties
// result (no properties) passes through unchanged.
func TestNormalizeGetProperties_EmptyResult(t *testing.T) {
	input := mustJSON(map[string]any{
		"id":     5,
		"result": map[string]any{"result": []any{}},
	})

	output := normalizeCDPResponse(input)
	if string(output) != string(input) {
		t.Errorf("empty result should pass through unchanged")
	}
}

// TestNormalizeGetProperties_NoModificationReturnOriginal verifies that
// when no modification happens, the original byte slice is returned
// (not a re-serialized copy).
func TestNormalizeGetProperties_NoModificationReturnOriginal(t *testing.T) {
	// An event — should pass through without any allocation
	input := []byte(`{"method":"Page.loadEventFired","params":{"timestamp":123}}`)
	output := normalizeCDPResponse(input)
	if string(output) != string(input) {
		t.Errorf("event should pass through byte-identical")
	}
}
