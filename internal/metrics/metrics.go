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

// Package metrics defines the operator's custom Prometheus metrics. They are
// served on the controller-runtime metrics endpoint alongside the standard
// Go runtime and controller metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Outcome values used for the reconciliations counter.
const (
	OutcomeSuccess  = "success"
	OutcomeError    = "error"
	OutcomeTerminal = "terminal"
)

var (
	// ReconciliationsTotal counts reconciliations by resource kind and
	// outcome: success (server state matches), error (transient failure,
	// retried with backoff) and terminal (condition False, needs operator
	// attention).
	ReconciliationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keycloak_operator_reconciliations_total",
			Help: "Reconciliations by resource kind and outcome (success, error, terminal).",
		},
		[]string{"kind", "outcome"},
	)

	// ReconcileDuration observes how long a full reconciliation takes, by
	// resource kind.
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "keycloak_operator_reconcile_duration_seconds",
			Help:    "Duration of resource reconciliations by kind.",
			Buckets: prometheus.ExponentialBuckets(0.005, 2, 14),
		},
		[]string{"kind"},
	)

	// DriftCorrectionsTotal counts server-side updates issued because the
	// Keycloak server state diverged from the declared state.
	DriftCorrectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keycloak_operator_drift_corrections_total",
			Help: "Server-side updates issued to correct drift, by resource kind.",
		},
		[]string{"kind"},
	)

	// ConnectionUp reports whether the operator can currently authenticate
	// against a Keycloak server.
	ConnectionUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "keycloak_operator_connection_up",
			Help: "Whether the operator can authenticate against the Keycloak server (1 = up, 0 = down).",
		},
		[]string{"namespace", "connection"},
	)

	// ServerInfo exposes the version of each connected Keycloak server as an
	// info-style metric with value 1.
	ServerInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "keycloak_operator_server_info",
			Help: "Keycloak server version per connection, always 1.",
		},
		[]string{"namespace", "connection", "version"},
	)

	// AdminRequestsTotal counts Admin API requests by connection, HTTP
	// method and response status class (2xx, 4xx, 5xx).
	AdminRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "keycloak_operator_admin_requests_total",
			Help: "Keycloak Admin API requests by connection, method and status class.",
		},
		[]string{"connection", "method", "code"},
	)

	// AdminRequestDuration observes Admin API request latency by connection
	// and HTTP method.
	AdminRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "keycloak_operator_admin_request_duration_seconds",
			Help:    "Keycloak Admin API request duration by connection and method.",
			Buckets: prometheus.ExponentialBuckets(0.005, 2, 14),
		},
		[]string{"connection", "method"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		ReconciliationsTotal,
		ReconcileDuration,
		DriftCorrectionsTotal,
		ConnectionUp,
		ServerInfo,
		AdminRequestsTotal,
		AdminRequestDuration,
	)
}
