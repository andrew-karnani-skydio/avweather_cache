# avweather-cache Helm Chart

A Helm chart for deploying the Aviation Weather Cache Service on Kubernetes.

## Introduction

This chart deploys avweather-cache, a service that caches METAR data from aviationweather.gov and provides a REST API for querying the data with multiple output formats (JSON, CSV, YAML).

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- (Optional) Prometheus Operator for ServiceMonitor support

## Installation

### Add Helm repository

```bash
helm repo add avweather-cache https://akarnani.github.io/avweather_cache
helm repo update
```

### Install the chart

```bash
helm install my-cache avweather-cache/avweather-cache
```

### Install with custom values

```bash
helm install my-cache avweather-cache/avweather-cache \
  --set config.cache.updateInterval=10m \
  --set resources.limits.memory=1Gi
```

### Install from local chart

```bash
helm install my-cache ./deploy/charts/avweather-cache
```

## Uninstallation

```bash
helm uninstall my-cache
```

## Configuration

The following table lists the configurable parameters of the chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Container image repository | `akarnani/avweather-cache` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (defaults to chart appVersion) | `""` |
| `service.type` | Kubernetes service type | `LoadBalancer` |
| `service.port` | Service port | `8080` |
| `config.server.port` | Application server port | `8080` |
| `config.cache.updateInterval` | Cache refresh interval (Go duration) | `"5m"` |
| `config.cache.sourceUrl` | METAR data source URL | `"https://aviationweather.gov/data/cache/metars.cache.xml.gz"` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |
| `metrics.enabled` | Enable Prometheus metrics | `true` |
| `metrics.serviceMonitor.enabled` | Create Prometheus ServiceMonitor | `true` |
| `metrics.serviceMonitor.interval` | Scrape interval | `30s` |
| `metrics.serviceMonitor.labels` | Additional ServiceMonitor labels | `{prometheus: kube-prometheus}` |
| `livenessProbe.enabled` | Enable liveness probe | `true` |
| `readinessProbe.enabled` | Enable readiness probe | `true` |
| `podSecurityContext.runAsUser` | User ID for pod | `65532` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |

### Service Types

The chart defaults to `LoadBalancer` which works well for local development (Docker Desktop, Rancher Desktop) and cloud providers with load balancer support.

#### LoadBalancer (default)
Exposes the service externally. Works on:
- **Rancher Desktop / Docker Desktop**: Service accessible at `localhost:8080` and `<host-ip>:8080`
- **Cloud providers (AWS, GCP, Azure)**: Gets a public IP address
- **Local clusters without LB support**: Will remain in "Pending" state

```bash
# Default - no changes needed
helm install my-cache avweather-cache/avweather-cache
```

#### ClusterIP
Only accessible within the cluster. Use for production clusters with Ingress:

```bash
helm install my-cache avweather-cache/avweather-cache \
  --set service.type=ClusterIP
```

Then access via port-forward: `kubectl port-forward svc/my-cache-avweather-cache 8080:8080`

#### NodePort
Exposes service on each node's IP at a high port (30000-32767):

```bash
helm install my-cache avweather-cache/avweather-cache \
  --set service.type=NodePort
```

### Example configurations

#### Faster cache updates

```bash
helm install my-cache avweather-cache/avweather-cache \
  --set config.cache.updateInterval=2m
```

#### Without ServiceMonitor (if Prometheus Operator not installed)

```bash
helm install my-cache avweather-cache/avweather-cache \
  --set metrics.serviceMonitor.enabled=false
```

The pod will still have Prometheus annotations for scraping:
- `prometheus.io/scrape: "true"`
- `prometheus.io/port: "8080"`
- `prometheus.io/path: "/metrics"`

#### Higher resource limits

```bash
helm install my-cache avweather-cache/avweather-cache \
  --set resources.limits.cpu=1 \
  --set resources.limits.memory=1Gi \
  --set resources.requests.cpu=250m \
  --set resources.requests.memory=256Mi
```

## Prometheus Integration

This chart supports two methods of Prometheus integration:

### 1. ServiceMonitor (Prometheus Operator)

If you have the Prometheus Operator installed, the chart creates a ServiceMonitor resource automatically:

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      prometheus: kube-prometheus  # Match your Prometheus selector
```

### 2. Pod Annotations (Standard Prometheus)

If you're using standard Prometheus scraping, disable ServiceMonitor and rely on pod annotations:

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: false

podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

## Accessing the Service

### Port-forward to local machine

```bash
kubectl port-forward svc/my-cache-avweather-cache 8080:8080
```

Then access:
- API: http://localhost:8080/api/metar?stations=KJFK
- Metrics: http://localhost:8080/metrics
- Web UI: http://localhost:8080/

### Get pod logs

```bash
kubectl logs -f deployment/my-cache-avweather-cache
```

## Key Metrics

The service exposes the following Prometheus metrics:

- `avweather_data_pull_age_seconds` - Age of cached data since last update
- `avweather_cached_stations` - Number of stations currently cached
- `avweather_api_requests_total` - Total API requests by endpoint and status
- `avweather_api_request_duration_seconds` - API request latency histogram

## Security

- Container runs as non-root user (UID 65532)
- All Linux capabilities dropped
- Read-only root filesystem (disabled for now, can be enabled)
- Uses distroless base image for minimal attack surface

## Troubleshooting

### ServiceMonitor not working

Check if Prometheus Operator is installed:
```bash
kubectl get crd servicemonitors.monitoring.coreos.com
```

If not installed, disable ServiceMonitor:
```bash
helm upgrade my-cache avweather-cache/avweather-cache \
  --set metrics.serviceMonitor.enabled=false
```

### Pod not starting

Check logs:
```bash
kubectl describe pod -l app.kubernetes.io/name=avweather-cache
kubectl logs -l app.kubernetes.io/name=avweather-cache
```

Common issues:
- Invalid update interval format (must be Go duration like "5m", "1h30m")
- Network issues reaching aviationweather.gov
- Resource limits too low

### Cache not updating

Check the `avweather_data_pull_age_seconds` metric:
```bash
kubectl port-forward svc/my-cache-avweather-cache 8080:8080
curl http://localhost:8080/metrics | grep avweather_data_pull_age_seconds
```

If age is growing without bound, check logs for errors fetching from source URL.

## License

See main project repository for license information.
