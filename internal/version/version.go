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

// Package version carries build metadata injected at link time.
package version

import "runtime"

// Build information. The defaults describe a development build; release
// builds override them with -ldflags -X (see the Makefile and Dockerfile).
var (
	// Version is the semantic version of the binary (e.g. "v0.2.0").
	Version = "dev"
	// Commit is the short git commit the binary was built from.
	Commit = "unknown"
	// Date is the UTC build date in RFC 3339 format.
	Date = "unknown"
)

// String renders the one-line version banner printed by --version.
func String() string {
	return "keycloak-operator " + Version +
		" (commit " + Commit + ", built " + Date +
		", " + runtime.Version() + ")"
}
