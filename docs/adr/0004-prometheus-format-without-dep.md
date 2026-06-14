# 4. Prometheus-format metrics without `client_golang`

Date: 2026-06-14

## Status

Accepted.

## Context

The server is small enough that a full observability stack is overkill,
but we still want one well-known scrape endpoint that monitoring tools
(Prometheus, VictoriaMetrics, Grafana Agent, etc.) understand without
extra configuration.

The de facto standard is `github.com/prometheus/client_golang`. It is a
mature, comprehensive library — but it pulls in a large dependency
tree (currently 30+ transitive packages, including `golang/protobuf`,
`cespare/xxhash`, `beorn7/perks`, and a custom HTTP transport) and
introduces a global default registry that is awkward to test and
reason about.

The Prometheus text exposition format is also simple: any HTTP
endpoint that returns a body matching the
[v0.0.4 specification](https://prometheus.io/docs/instrumenting/exposition_formats/)
is scrape-compatible. We have at most a handful of metrics (cache
hits, misses, evictions, uptime) and adding the full client library
for those is overkill.

## Decision

We expose `/metrics` as a **hand-written text-format endpoint** that
emits the `dynamic_markdown_site_*` metric family. The endpoint lives
in `internal/server/metrics.go` and uses `fmt.Fprintf` against a
`strings.Builder`. No new dependencies are introduced.

The contract is:

- `Content-Type: text/plain; version=0.0.4; charset=utf-8`
- HELP and TYPE comments for every metric
- Numeric values as plain decimals

## Consequences

Positive:

- Zero new dependencies; the entire metrics surface is one 60-line
  file
- Trivially testable: assert on substrings of the response body
- The endpoint is not coupled to a particular Prometheus SDK version
- Easy to add custom metrics (e.g., per-route timings) without
  touching a global registry

Negative:

- We do not get label dimensions, histograms, or summaries for free
- If we need richer metrics in the future (per-route p99 latencies,
  high-cardinality labels), we will need to introduce the official
  client library or write our own histogram implementation
- We must remember to keep the text format valid; we add integration
  tests for the well-known metric names

## Alternatives considered

- **`prometheus/client_golang`** — heavyweight for our needs; adds
  ~30 transitive deps and a global default registry
- **OpenTelemetry** — much larger; not justified for a 3-line
  metrics surface
- **`prometheus/common`** alone — exports the format parsers, not
  emitters
