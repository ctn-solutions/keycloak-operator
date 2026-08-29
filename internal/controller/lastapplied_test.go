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
	"testing"
)

func TestEffectivePayloadRemovals(t *testing.T) {
	spec := map[string]any{"realm": "demo", "enabled": true}
	last := `{"realm":"demo","enabled":true,"displayName":"Old","registrationAllowed":true,"accessTokenLifespan":300,"attributes":{"a":"b"}}`

	payload, err := EffectivePayload(spec, last)
	if err != nil {
		t.Fatalf("EffectivePayload: %v", err)
	}
	if payload["displayName"] != "" {
		t.Errorf("string removal should reset to empty, got %v", payload["displayName"])
	}
	if payload["registrationAllowed"] != false {
		t.Errorf("bool removal should reset to false, got %v", payload["registrationAllowed"])
	}
	if _, reset := payload["accessTokenLifespan"]; reset {
		t.Errorf("numeric removal must not be reset, got %v", payload["accessTokenLifespan"])
	}
	if attrs, ok := payload["attributes"].(map[string]any); !ok || len(attrs) != 0 {
		t.Errorf("map removal should reset to empty map, got %v", payload["attributes"])
	}
	if payload["realm"] != "demo" || payload["enabled"] != true {
		t.Errorf("spec fields must be preserved, got %v", payload)
	}
}

func TestEffectivePayloadIsolatesSpec(t *testing.T) {
	spec := map[string]any{"realm": "demo", "smtpServer": map[string]any{"host": "smtp.example.com"}}
	payload, err := EffectivePayload(spec, "")
	if err != nil {
		t.Fatalf("EffectivePayload: %v", err)
	}
	// Mutating the payload (as secret injection does) must not touch the
	// spec snapshot that gets recorded in the last-applied annotation.
	smtp := payload["smtpServer"].(map[string]any)
	smtp["password"] = "injected-secret"
	if spec["smtpServer"].(map[string]any)["password"] != nil {
		t.Fatal("payload mutation leaked into the spec snapshot")
	}
}

func TestPayloadDiffers(t *testing.T) {
	cases := []struct {
		name    string
		remote  map[string]any
		payload map[string]any
		want    bool
	}{
		{"identical", map[string]any{"displayName": "Demo"}, map[string]any{"displayName": "Demo"}, false},
		{"changed", map[string]any{"displayName": "Demo"}, map[string]any{"displayName": "Other"}, true},
		{"missing key", map[string]any{}, map[string]any{"redirectUris": []any{"x"}}, true},
		// A zero-value want matches an absent remote key: servers omit
		// cleared fields, and absent is the goal of a removal.
		{"zero want, absent remote", map[string]any{}, map[string]any{"displayName": ""}, false},
		{"false want, absent remote", map[string]any{}, map[string]any{"notBefore": false}, false},
		{"empty list want, absent remote", map[string]any{}, map[string]any{"webOrigins": []any{}}, false},
		{"zero want, non-zero remote", map[string]any{"displayName": "Demo"}, map[string]any{"displayName": ""}, true},
	}

	for _, tc := range cases {
		if got := PayloadDiffers(tc.remote, tc.payload); got != tc.want {
			t.Errorf("%s: PayloadDiffers = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMergePayloadRemovesNil(t *testing.T) {
	remote := map[string]any{"realm": "demo", "displayName": "Demo", "attributes": map[string]any{"a": "b"}}
	merged := MergePayload(remote, map[string]any{"displayName": nil, "enabled": true})
	if _, present := merged["displayName"]; present {
		t.Error("nil payload value should remove the key")
	}
	if merged["enabled"] != true {
		t.Error("non-nil payload value should win")
	}
	if merged["attributes"].(map[string]any)["a"] != "b" {
		t.Error("unmanaged keys must be preserved")
	}
}

func TestDeepCopyMap(t *testing.T) {
	original := map[string]any{
		"nested": map[string]any{"a": "b"},
		"list":   []any{"x", map[string]any{"y": "z"}},
	}
	copied := deepCopyMap(original)
	copied["nested"].(map[string]any)["a"] = "changed"
	copied["list"].([]any)[1].(map[string]any)["y"] = "changed"
	if original["nested"].(map[string]any)["a"] != "b" {
		t.Error("nested map copy is not deep")
	}
	if original["list"].([]any)[1].(map[string]any)["y"] != "z" {
		t.Error("nested list copy is not deep")
	}
}
