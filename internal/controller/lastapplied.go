/*
Copyright 2026 CTN Solutions

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"encoding/json"
	"fmt"
)

// EffectivePayload computes the payload the operator should enforce from the
// current spec and the last-applied annotation.
//
// Fields present in the last-applied payload but absent from the current spec
// are reset: strings, booleans, arrays and objects are set to their zero
// value so the server clears them, and Keycloak interprets zero values as
// its defaults for most fields. Numeric fields are left unchanged on removal
// because the Admin API cannot distinguish an unset number from zero.
func EffectivePayload(specMap map[string]any, lastAppliedJSON string) (map[string]any, error) {
	// Deep-copy: callers mutate the payload (secret injection, managed
	// marker) and the spec snapshot must stay pristine for the last-applied
	// annotation.
	payload := deepCopyMap(specMap)
	if lastAppliedJSON == "" {
		return payload, nil
	}
	var last map[string]any
	if err := json.Unmarshal([]byte(lastAppliedJSON), &last); err != nil {
		return nil, fmt.Errorf("decode last-applied annotation: %w", err)
	}
	for k, previous := range last {
		if _, ok := payload[k]; !ok {
			if zero, resettable := zeroFor(previous); resettable {
				payload[k] = zero
			}
		}
	}
	return payload, nil
}

// zeroFor returns the reset value for a previously applied value. Numbers are
// not resettable.
func zeroFor(previous any) (any, bool) {
	switch previous.(type) {
	case string:
		return "", true
	case bool:
		return false, true
	case []any:
		return []any{}, true
	case map[string]any:
		return map[string]any{}, true
	default:
		return nil, false
	}
}

// MergePayload overlays the payload onto the remote representation. A nil
// payload value removes the key from the remote map; any other value wins.
// The remote map is mutated in place and returned.
func MergePayload(remote, payload map[string]any) map[string]any {
	for k, v := range payload {
		if v == nil {
			delete(remote, k)
			continue
		}
		remote[k] = v
	}
	return remote
}

// PayloadDiffers reports whether the remote representation differs from the
// payload on any key the payload manages. Both sides are normalised through
// JSON so numbers and nested structures compare consistently.
//
// A zero-value payload entry ("", false, [], {}) matches a remote key that is
// absent as well as one set to the same zero value: servers commonly omit
// cleared fields from their representations, and absent is exactly the state
// a removal aims for. Without this, clearing a field the server omits would
// produce an update that never converges.
func PayloadDiffers(remote, payload map[string]any) bool {
	for k, want := range payload {
		got, exists := remote[k]
		if want == nil {
			if exists && got != nil {
				return true
			}
			continue
		}
		if isZeroValue(want) {
			if exists && got != nil && !managedValueEqual(want, got) {
				return true
			}
			continue
		}
		if !exists || got == nil {
			return true
		}
		if !managedValueEqual(want, got) {
			return true
		}
	}
	return false
}

// managedValueEqual compares a desired value against the remote value. Maps
// are compared on the desired keys only: the server routinely adds its own
// entries inside maps the spec also manages (attributes, broker config), and
// those keys are not ours to enforce. String values that match the Keycloak
// secret mask are treated as equal to anything: the server masks sensitive
// config values in GET responses, so their real value cannot be verified.
func managedValueEqual(want, got any) bool {
	if wantMap, ok := want.(map[string]any); ok {
		gotMap, ok := got.(map[string]any)
		if !ok {
			return false
		}
		return !PayloadDiffers(gotMap, wantMap)
	}
	if wantList, ok := want.([]any); ok {
		gotList, ok := got.([]any)
		if !ok || len(wantList) != len(gotList) {
			return false
		}
		// Order-insensitive: each desired element must match a distinct
		// remote element. Servers augment list elements (for example mapper
		// ids), so elements are compared on the desired keys only.
		used := make([]bool, len(gotList))
		for _, wantItem := range wantList {
			matched := false
			for i, gotItem := range gotList {
				if used[i] {
					continue
				}
				if managedValueEqual(wantItem, gotItem) {
					used[i] = true
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	}
	if gotStr, ok := got.(string); ok && gotStr == keycloakMaskedValue {
		return true
	}
	return jsonEqual(want, got)
}

// keycloakMaskedValue is what the Keycloak Admin API returns for sensitive
// configuration values it does not disclose.
const keycloakMaskedValue = "**********"

// isZeroValue reports whether v is one of the reset values produced by
// EffectivePayload for removed fields.
func isZeroValue(v any) bool {
	switch v := v.(type) {
	case string:
		return v == ""
	case bool:
		return !v
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}

// deepCopyMap returns a deep copy of a JSON-style map so callers can mutate
// nested values without affecting the original snapshot.
func deepCopyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch value := v.(type) {
	case map[string]any:
		return deepCopyMap(value)
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = deepCopyValue(item)
		}
		return out
	default:
		return v
	}
}

// jsonEqual compares two values through a JSON round-trip so that int and
// float64 representations of the same number compare equal.
func jsonEqual(a, b any) bool {
	aRaw, aErr := json.Marshal(a)
	bRaw, bErr := json.Marshal(b)
	if aErr != nil || bErr != nil {
		return false
	}
	var aNorm, bNorm any
	if json.Unmarshal(aRaw, &aNorm) != nil || json.Unmarshal(bRaw, &bNorm) != nil {
		return false
	}
	return string(aRaw) == string(bRaw) || deepEqualJSON(aNorm, bNorm)
}

func deepEqualJSON(a, b any) bool {
	aRaw, _ := json.Marshal(a)
	bRaw, _ := json.Marshal(b)
	return string(aRaw) == string(bRaw)
}
