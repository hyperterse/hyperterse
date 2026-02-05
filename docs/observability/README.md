# Observability Setup Guide

This guide explains how to set up and use the observability infrastructure for Hyperterse.

## Architecture

Hyperterse uses OpenTelemetry Collector as a telemetry gateway:

```
Hyperterse App → OTel Collector → [Jaeger (traces) / Prometheus (metrics)]
```

This decouples the application from backends, making it easy to switch or add backends without code changes.

## Quick Start

1. **Start the observability stack:**

```bash
docker-compose -f docker-compose.observability.yml up -d
```

2. **Configure the app to export telemetry:**

Set environment variables:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_SERVICE_NAME=hyperterse
export OTEL_SERVICE_VERSION=1.0.0
```

3. **Run Hyperterse:**

The app will automatically export traces and metrics to the OTel Collector.

## Viewing Traces

Open Jaeger UI: http://localhost:16686

- Search for traces by service name (`hyperterse`)
- Filter by operation, tags, or time range
- View detailed trace spans and timing

## Viewing Metrics

Open Prometheus UI: http://localhost:9090

- Query metrics using PromQL
- View metrics exposed by OTel Collector on port 8889
- Example queries:
  - `rate(http_requests_total[5m])` - Request rate
  - `histogram_quantile(0.95, http_request_duration_seconds_bucket)` - P95 latency

## Optional: Grafana Dashboards

Start Grafana (included in docker-compose):

```bash
docker-compose -f docker-compose.observability.yml --profile optional up -d grafana
```

Access Grafana: http://localhost:3000
- Default credentials: admin/admin
- Add Prometheus as data source: http://prometheus:9090
- Add Jaeger as data source: http://jaeger:16686

## Configuration

### OTel Collector

Edit `otel-collector-config.yaml` to:
- Add/remove exporters
- Configure processors (batch, resource, etc.)
- Adjust sampling rates

### Prometheus

Edit `prometheus-config.yml` to:
- Add scrape targets
- Configure scrape intervals
- Set up alerting rules

## Production Considerations

1. **Use TLS:** Update `otel-collector-config.yaml` and `otel.go` to use TLS for OTLP endpoints
2. **Sampling:** Adjust trace sampling in `otel.go` (currently `AlwaysSample()`)
3. **Resource Limits:** Set appropriate memory/CPU limits in docker-compose
4. **Persistence:** Configure persistent storage for Prometheus and Grafana
5. **High Availability:** Set up multiple OTel Collector instances behind a load balancer

## Troubleshooting

**Traces not appearing in Jaeger:**
- Check OTel Collector logs: `docker logs otel-collector`
- Verify endpoint: `OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317`
- Check Jaeger is receiving data: `docker logs jaeger`

**Metrics not appearing in Prometheus:**
- Check OTel Collector metrics endpoint: `curl http://localhost:8889/metrics`
- Verify Prometheus is scraping: Check Prometheus targets page
- Check Prometheus config: `docker exec prometheus cat /etc/prometheus/prometheus.yml`
