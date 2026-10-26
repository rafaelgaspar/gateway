// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package translator

import (
	"testing"
	"time"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/envoyproxy/gateway/internal/ir"
)

func TestBuildAdaptiveConcurrencyProtoDefaultsConcurrencyUpdateInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval *metav1.Duration
	}{
		{
			name: "omitted",
		},
		{
			name:     "zero",
			interval: &metav1.Duration{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := buildAdaptiveConcurrencyProto(&ir.AdaptiveConcurrency{
				ConcurrencyUpdateInterval: tt.interval,
			})

			params := ac.GetGradientControllerConfig().GetConcurrencyLimitParams()
			require.NotNil(t, params)
			require.Equal(t, defaultAdaptiveConcurrencyUpdateInterval, params.GetConcurrencyUpdateInterval().AsDuration())
		})
	}
}

func TestBuildAdaptiveConcurrencyProtoUsesConfiguredConcurrencyUpdateInterval(t *testing.T) {
	ac := buildAdaptiveConcurrencyProto(&ir.AdaptiveConcurrency{
		SampleAggregatePercentile:      new(float64(90)),
		MaxConcurrencyLimit:            new(uint32(500)),
		ConcurrencyUpdateInterval:      &metav1.Duration{Duration: 200 * time.Millisecond},
		MinRTTCalcInterval:             &metav1.Duration{Duration: 30 * time.Second},
		FixedMinRTT:                    &metav1.Duration{Duration: 10 * time.Millisecond},
		RequestCount:                   new(uint32(100)),
		Jitter:                         new(float64(10)),
		MinConcurrency:                 new(uint32(5)),
		Buffer:                         new(float64(30)),
		ConcurrencyLimitExceededStatus: new(int32(429)),
	})

	params := ac.GetGradientControllerConfig().GetConcurrencyLimitParams()
	require.NotNil(t, params)
	require.Equal(t, 200*time.Millisecond, params.GetConcurrencyUpdateInterval().AsDuration())
	require.Equal(t, uint32(500), params.GetMaxConcurrencyLimit().GetValue())
	require.Equal(t, float64(90), ac.GetGradientControllerConfig().GetSampleAggregatePercentile().GetValue())

	minRTT := ac.GetGradientControllerConfig().GetMinRttCalcParams()
	require.Equal(t, 30*time.Second, minRTT.GetInterval().AsDuration())
	require.Equal(t, 10*time.Millisecond, minRTT.GetFixedValue().AsDuration())
	require.Equal(t, uint32(100), minRTT.GetRequestCount().GetValue())
	require.Equal(t, float64(10), minRTT.GetJitter().GetValue())
	require.Equal(t, uint32(5), minRTT.GetMinConcurrency().GetValue())
	require.Equal(t, float64(30), minRTT.GetBuffer().GetValue())
	require.Equal(t, int32(429), int32(ac.GetConcurrencyLimitExceededStatus().GetCode()))
}

func TestAdaptiveConcurrencyPatchHCM(t *testing.T) {
	filter := &adaptiveConcurrency{}

	t.Run("nil hcm", func(t *testing.T) {
		require.EqualError(t, filter.patchHCM(nil, &ir.HTTPListener{}), "hcm is nil")
	})
	t.Run("nil listener", func(t *testing.T) {
		require.EqualError(t, filter.patchHCM(&hcmv3.HttpConnectionManager{}, nil), "ir listener is nil")
	})
	t.Run("no adaptive concurrency", func(t *testing.T) {
		mgr := &hcmv3.HttpConnectionManager{}
		require.NoError(t, filter.patchHCM(mgr, &ir.HTTPListener{Routes: []*ir.HTTPRoute{{}}}))
		require.Empty(t, mgr.HttpFilters)
	})
	t.Run("adds disabled filter once", func(t *testing.T) {
		ac := &ir.AdaptiveConcurrency{Name: "backendtrafficpolicy/default/policy"}
		route := &ir.HTTPRoute{Traffic: &ir.TrafficFeatures{AdaptiveConcurrency: ac}}
		mgr := &hcmv3.HttpConnectionManager{}
		listener := &ir.HTTPListener{Routes: []*ir.HTTPRoute{route, route}}
		require.NoError(t, filter.patchHCM(mgr, listener))
		require.Len(t, mgr.HttpFilters, 1)
		require.True(t, mgr.HttpFilters[0].Disabled)
		require.Equal(t, "envoy.filters.http.adaptive_concurrency/backendtrafficpolicy/default/policy", mgr.HttpFilters[0].Name)
	})
}

func TestAdaptiveConcurrencyPatchRoute(t *testing.T) {
	filter := &adaptiveConcurrency{}

	t.Run("nil route", func(t *testing.T) {
		require.EqualError(t, filter.patchRoute(nil, &ir.HTTPRoute{}, nil), "xds route is nil")
	})
	t.Run("nil ir route", func(t *testing.T) {
		require.EqualError(t, filter.patchRoute(&routev3.Route{}, nil, nil), "ir route is nil")
	})
	t.Run("no adaptive concurrency", func(t *testing.T) {
		route := &routev3.Route{}
		require.NoError(t, filter.patchRoute(route, &ir.HTTPRoute{}, nil))
		require.Empty(t, route.TypedPerFilterConfig)
	})
	t.Run("enables filter", func(t *testing.T) {
		irRoute := &ir.HTTPRoute{
			Traffic: &ir.TrafficFeatures{
				AdaptiveConcurrency: &ir.AdaptiveConcurrency{Name: "backendtrafficpolicy/default/policy"},
			},
		}
		route := &routev3.Route{}
		require.NoError(t, filter.patchRoute(route, irRoute, nil))
		require.Contains(t, route.TypedPerFilterConfig, "envoy.filters.http.adaptive_concurrency/backendtrafficpolicy/default/policy")
		err := filter.patchRoute(route, irRoute, nil)
		require.ErrorContains(t, err, "route already contains filter config: envoy.filters.http.adaptive_concurrency/backendtrafficpolicy/default/policy")
	})
}

func TestRouteContainsAdaptiveConcurrency(t *testing.T) {
	require.False(t, routeContainsAdaptiveConcurrency(nil))
	require.False(t, routeContainsAdaptiveConcurrency(&ir.HTTPRoute{}))
	require.True(t, routeContainsAdaptiveConcurrency(&ir.HTTPRoute{
		Traffic: &ir.TrafficFeatures{AdaptiveConcurrency: &ir.AdaptiveConcurrency{}},
	}))
}

func TestAdaptiveConcurrencyPatchResources(t *testing.T) {
	require.NoError(t, (&adaptiveConcurrency{}).patchResources(nil, nil))
}
