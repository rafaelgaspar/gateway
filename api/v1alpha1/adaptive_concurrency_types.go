// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package v1alpha1

import (
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// AdaptiveConcurrency configures Envoy's adaptive concurrency HTTP filter, which
// dynamically adjusts the allowed request concurrency based on sampled latencies.
// This protects backends from overload by shedding load when latency increases.
//
// Each BackendTrafficPolicy gets its own filter instance so concurrency state is
// not shared across unrelated routes. Empty configuration uses Envoy's documented
// gradient-controller defaults, except concurrencyUpdateInterval, which Envoy
// requires and Envoy Gateway defaults to 100ms.
//
// +kubebuilder:validation:XValidation:rule="!has(self.gradientController) || !has(self.gradientController.concurrencyLimitParams) || !has(self.gradientController.concurrencyLimitParams.concurrencyUpdateInterval) || duration(self.gradientController.concurrencyLimitParams.concurrencyUpdateInterval) > duration('0s')",message="concurrencyUpdateInterval must be greater than 0s"
// +kubebuilder:validation:XValidation:rule="!has(self.gradientController) || !has(self.gradientController.minRTTCalcParams) || !has(self.gradientController.minRTTCalcParams.interval) || duration(self.gradientController.minRTTCalcParams.interval) > duration('0s') || has(self.gradientController.minRTTCalcParams.fixedMinRTT)",message="fixedMinRTT is required when minRTTCalcParams.interval is 0s"
type AdaptiveConcurrency struct {
	// GradientController configures the gradient-based concurrency controller.
	// This is the core algorithm that adjusts the concurrency limit based on
	// latency measurements.
	//
	// +optional
	GradientController *GradientController `json:"gradientController,omitempty"`

	// ConcurrencyLimitExceededStatus sets the HTTP status code returned to
	// downstream clients when the concurrency limit is exceeded.
	// Defaults to 503 (Service Unavailable).
	//
	// +optional
	// +kubebuilder:default=503
	// +kubebuilder:validation:Minimum=400
	// +kubebuilder:validation:Maximum=599
	ConcurrencyLimitExceededStatus *int32 `json:"concurrencyLimitExceededStatus,omitempty"`
}

// GradientController configures the gradient-based concurrency controller algorithm.
// The controller periodically samples request latencies and adjusts the concurrency
// limit using a gradient calculated from the current vs minimum RTT.
type GradientController struct {
	// SampleAggregatePercentile specifies the percentile to use when
	// summarizing aggregated latency samples. Defaults to 50 (p50).
	//
	// +optional
	// +kubebuilder:default=50
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	SampleAggregatePercentile *float64 `json:"sampleAggregatePercentile,omitempty"`

	// ConcurrencyLimitParams controls the periodic concurrency limit
	// recalculation.
	//
	// +optional
	ConcurrencyLimitParams *ConcurrencyLimitParams `json:"concurrencyLimitParams,omitempty"`

	// MinRTTCalcParams controls the periodic minRTT recalculation.
	//
	// +optional
	MinRTTCalcParams *MinRTTCalcParams `json:"minRTTCalcParams,omitempty"`
}

// ConcurrencyLimitParams controls parameters for the periodic recalculation
// of the concurrency limit from sampled request latencies.
type ConcurrencyLimitParams struct {
	// MaxConcurrencyLimit sets the upper bound on the concurrency limit.
	// Defaults to 1000.
	//
	// +optional
	// +kubebuilder:default=1000
	// +kubebuilder:validation:Minimum=1
	MaxConcurrencyLimit *uint32 `json:"maxConcurrencyLimit,omitempty"`

	// ConcurrencyUpdateInterval is the period of time between
	// recalculations of the concurrency limit.
	// Defaults to 100ms. Envoy requires this field to be greater than 0s.
	//
	// +optional
	// +kubebuilder:default="100ms"
	ConcurrencyUpdateInterval *gwapiv1.Duration `json:"concurrencyUpdateInterval,omitempty"`
}

// MinRTTCalcParams controls parameters for the periodic minRTT recalculation.
type MinRTTCalcParams struct {
	// Interval is the time interval between recalculating the minimum RTT.
	// Defaults to 30s. Setting this to 0s disables dynamic minRTT sampling
	// (FixedMinRTT is required in that case).
	//
	// +optional
	// +kubebuilder:default="30s"
	Interval *gwapiv1.Duration `json:"interval,omitempty"`

	// FixedMinRTT sets a fixed minimum RTT value instead of dynamically
	// sampling it. This is required when Interval is set to 0s.
	//
	// +optional
	FixedMinRTT *gwapiv1.Duration `json:"fixedMinRTT,omitempty"`

	// RequestCount is the number of requests to aggregate during the minRTT
	// recalculation window. Defaults to 50.
	//
	// +optional
	// +kubebuilder:default=50
	// +kubebuilder:validation:Minimum=1
	RequestCount *uint32 `json:"requestCount,omitempty"`

	// Jitter adds a randomized delay to the start of the minRTT calculation
	// window, expressed as a percentage of the interval. Defaults to 15%.
	//
	// +optional
	// +kubebuilder:default=15
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Jitter *float64 `json:"jitter,omitempty"`

	// MinConcurrency sets the concurrency limit used while measuring the
	// minRTT. Defaults to 3.
	//
	// +optional
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	MinConcurrency *uint32 `json:"minConcurrency,omitempty"`

	// Buffer is the amount added to the measured minRTT to add stability
	// to the concurrency limit, expressed as a percentage of the measured
	// value. Defaults to 25%.
	//
	// +optional
	// +kubebuilder:default=25
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Buffer *float64 `json:"buffer,omitempty"`
}
