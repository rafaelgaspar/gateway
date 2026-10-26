Added `adaptiveConcurrency` to `BackendTrafficPolicy` so Envoy can dynamically
adjust request concurrency from sampled latencies and shed load when backends
slow down. The policy attaches to `Gateway`, `ListenerSet`, `HTTPRoute`, or
`GRPCRoute` and is translated to Envoy's adaptive concurrency HTTP filter.
