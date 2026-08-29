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
// are set to nil, which the merge step turns into a removal on the server so
// Keycloak falls back to its default. This gives declarative field removal
// without clobbering fields the spec never managed.
func EffectivePayload(specMap map[string]any, lastAppliedJSON string) (map[string]any, error) {
	payload := make(map[string]any, len(specMap))
	for k, v := range specMap {
		payload[k] = v
	}
	if lastAppliedJSON == "" {
		return payload, nil
	}
	var last map[string]any
	if err := json.Unmarshal([]byte(lastAppliedJSON), &last); err != nil {
		return nil, fmt.Errorf("decode last-applied annotation: %w", err)
	}
	for k := range last {
		if _, ok := payload[k]; !ok {
			payload[k] = nil
		}
	}
	return payload, nil
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
// JSON so numbers and nested structures compare consistently. A nil payload
// value requires the key to be absent (or null) on the remote side.
func PayloadDiffers(remote, payload map[string]any) bool {
	for k, want := range payload {
		got, exists := remote[k]
		if want == nil {
			if exists && got != nil {
				return true
			}
			continue
		}
		if !exists || got == nil {
			return true
		}
		if !jsonEqual(want, got) {
			return true
		}
	}
	return false
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
