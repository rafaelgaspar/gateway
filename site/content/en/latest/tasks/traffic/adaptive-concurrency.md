---
title: "Adaptive Concurrency"
---

[Envoy Adaptive Concurrency][] dynamically limits in-flight requests based on
sampled latencies. Envoy measures a baseline round-trip time, then lowers the
concurrency limit when latency rises so overloaded backends can recover.

Envoy Gateway exposes this through [BackendTrafficPolicy][]. Attach the policy
to a [Gateway][], [HTTPRoute][], or [GRPCRoute][]. Each policy gets its own
filter instance so concurrency state is not shared across unrelated routes.

## Prerequisites

{{< boilerplate prerequisites >}}

## Configuration

The following policy enables adaptive concurrency on the example HTTPRoute with
Envoy's documented gradient-controller defaults, except a 200ms update interval
and a p90 sample percentile.

{{< tabpane text=true >}}
{{% tab header="Apply from stdin" %}}

```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: adaptive-concurrency-for-route
spec:
  targetRefs:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: backend
  adaptiveConcurrency:
    concurrencyLimitExceededStatus: 503
    gradientController:
      sampleAggregatePercentile: 90
      concurrencyLimitParams:
        maxConcurrencyLimit: 1000
        concurrencyUpdateInterval: 200ms
      minRTTCalcParams:
        interval: 30s
        requestCount: 50
        jitter: 15
        minConcurrency: 3
        buffer: 25
EOF
```

{{% /tab %}}
{{% tab header="Apply from file" %}}
Save and apply the following resource to your cluster:

```yaml
---
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: adaptive-concurrency-for-route
spec:
  targetRefs:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: backend
  adaptiveConcurrency:
    concurrencyLimitExceededStatus: 503
    gradientController:
      sampleAggregatePercentile: 90
      concurrencyLimitParams:
        maxConcurrencyLimit: 1000
        concurrencyUpdateInterval: 200ms
      minRTTCalcParams:
        interval: 30s
        requestCount: 50
        jitter: 15
        minConcurrency: 3
        buffer: 25
```

{{% /tab %}}
{{< /tabpane >}}

When the measured latency exceeds the sampled minimum RTT (plus `buffer`), Envoy
reduces the concurrency limit. Requests that exceed the limit receive
`concurrencyLimitExceededStatus` (503 by default) instead of being forwarded.

`concurrencyUpdateInterval` is required by Envoy and must be greater than `0s`.
If omitted, Envoy Gateway defaults it to `100ms`. Setting `minRTTCalcParams.interval`
to `0s` disables dynamic minRTT sampling and requires `fixedMinRTT`.

## Next Steps

- [Circuit Breakers](../circuit-breaker)
- [Admission Control](../../../api/extension_types#admissioncontrol)
- [BackendTrafficPolicy API](../../../api/extension_types#backendtrafficpolicy)

[Envoy Adaptive Concurrency]: https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/adaptive_concurrency_filter
[BackendTrafficPolicy]: ../../../api/extension_types#backendtrafficpolicy
[Gateway]: https://gateway-api.sigs.k8s.io/reference/api-types/gateway/
[HTTPRoute]: https://gateway-api.sigs.k8s.io/reference/api-types/httproute/
[GRPCRoute]: https://gateway-api.sigs.k8s.io/reference/api-types/grpcroute/
