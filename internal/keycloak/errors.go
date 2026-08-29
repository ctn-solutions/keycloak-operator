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

// Package keycloak implements a focused client for the Keycloak Admin API.
package keycloak

import (
	"errors"
	"fmt"
)

// Sentinel errors the reconcilers branch on.
var (
	// ErrNotFound is returned when the server reports 404 for a resource.
	ErrNotFound = errors.New("resource not found")
	// ErrConflict is returned when the server reports 409.
	ErrConflict = errors.New("resource conflict")
	// ErrAuth is returned when authentication fails, including after a
	// token refresh retry.
	ErrAuth = errors.New("authentication failed")
)

// APIError carries the details of an unexpected Admin API response.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("keycloak admin api %s %s: status %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}
