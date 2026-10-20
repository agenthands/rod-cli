package cdpproxy

import (
	"encoding/json"
)

// normalizeCDPResponse detects Runtime.getProperties responses and strips
// `value` from accessor (getter) properties to suppress property-getter
// triggering observable from the page.
//
// The heuristic is fail-safe: unparseable messages pass through unchanged.
// Only JSON-RPC responses (have "result", no "method") are inspected;
// events and requests pass through.
func normalizeCDPResponse(data []byte) []byte {
	// Peek at the message envelope to decide if this is a response.
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Method json.RawMessage `json:"method,omitempty"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return data // unparseable — pass through
	}
	// Only responses have "result" and no "method". Events and requests
	// have "method" set; skip those.
	if envelope.Result == nil || envelope.Method != nil {
		return data
	}

	// Try to parse the result as a Runtime.getProperties result shape:
	// {"result": [ ...property descriptors... ]}
	// Decode into a map so sibling keys (internalProperties,
	// privateProperties, exceptionDetails, ...) survive the rebuild.
	var outer map[string]json.RawMessage
	var props []json.RawMessage
	if err := json.Unmarshal(envelope.Result, &outer); err != nil {
		return data
	}
	if err := json.Unmarshal(outer["result"], &props); err != nil || props == nil {
		return data // not a getProperties response
	}

	// Iterate property descriptors and strip `value` from accessor properties.
	modified := false
	clean := make([]json.RawMessage, len(props))
	for i, rawProp := range props {
		var prop map[string]json.RawMessage
		if err := json.Unmarshal(rawProp, &prop); err != nil {
			clean[i] = rawProp
			continue
		}
		if _, hasGet := prop["get"]; hasGet {
			delete(prop, "value")
			b, err := json.Marshal(prop)
			if err != nil {
				clean[i] = rawProp
				continue
			}
			clean[i] = b
			modified = true
		} else {
			clean[i] = rawProp
		}
	}

	if !modified {
		return data
	}

	// Rebuild the full response.
	newProps, err := json.Marshal(clean)
	if err != nil {
		return data
	}
	outer["result"] = newProps
	newResult, err := json.Marshal(outer)
	if err != nil {
		return data
	}

	var full map[string]json.RawMessage
	if err := json.Unmarshal(data, &full); err != nil {
		return data
	}
	full["result"] = newResult
	out, err := json.Marshal(full)
	if err != nil {
		return data
	}
	return out
}
