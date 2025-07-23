# OAuth2 Server Helm Chart

This Helm chart deploys the OAuth2 Server application on a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.8+

## Installation

### Add the chart repository (if published)
```bash
helm repo add oauth2-server https://your-charts-repo.com
helm repo update
```

### Install from local directory
```bash
# Install with default values
helm install oauth2-server ./helm/oauth2-server

# Install with custom values
helm install oauth2-server ./helm/oauth2-server -f values-dev.yaml

# Install in specific namespace
helm install oauth2-server ./helm/oauth2-server -n oauth2 --create-namespace
```

### Upgrade
```bash
helm upgrade oauth2-server ./helm/oauth2-server -f values-prod.yaml
```

### Uninstall
```bash
helm uninstall oauth2-server
```

## Configuration

### Key Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `harrykodden/oauth2-server` |
| `image.tag` | Image tag | `latest` |
| `service.port` | Service port | `8080` |
| `ingress.enabled` | Enable ingress | `false` |
| `config.jwt.secret` | JWT signing secret | `""` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `autoscaling.enabled` | Enable HPA | `false` |

### Security Configuration

For production deployments, ensure:

1. Set a strong JWT secret
2. Enable TLS on ingress
3. Configure appropriate resource limits
4. Enable network policies if needed
5. Use non-root security context (enabled by default)

## Examples

### Development Deployment
```bash
helm install oauth2-server-dev ./helm/oauth2-server \
  --set ingress.enabled=true \
  --set config.jwt.secret="dev-secret" \
  --set image.tag="latest"
```

### Production Deployment
```bash
helm install oauth2-server-prod ./helm/oauth2-server \
  -f values-prod.yaml \
  --set config.jwt.secret="$(openssl rand -base64 32)"
```

### Sample deployment with public hostname
```bash
helm upgrade --install oauth2-server ./helm/oauth2-server 
-n oauth2-server --create-namespace --set config.server.baseUrl="https://oauth2-server.homelab.kodden.nl"
```