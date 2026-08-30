/*
Copyright 2026.

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

package version

import (
	"strings"
	"testing"
)

func TestStringContainsBuildMetadata(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = origVersion, origCommit, origDate })
	Version, Commit, Date = "v9.9.9", "abc1234", "2026-08-30T00:00:00Z"
	got := String()
	for _, want := range []string{"keycloak-operator v9.9.9", "commit abc1234", "built 2026-08-30"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
}

func TestDefaultsAreDevelopmentValues(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version = %q, want %q", Version, "dev")
	}
	if Commit != "unknown" || Date != "unknown" {
		t.Errorf("Commit/Date = %q/%q, want unknown/unknown", Commit, Date)
	}
}
