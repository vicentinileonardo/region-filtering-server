# Region filtering server

## Description

This repository is part of a project: **Carbon-aware workload scheduling in a multi-cloud environment**.

This is a simple server that filters cloud regions based on a **region of origin** and a **latency threshold**. 

Only Azure regions are supported at the moment.

## How to run

### Local deployment

```bash
go run main.go
```

Test the server by sending a request like the following:
```bash
curl -X POST http://localhost:8080/regions/eligible \
  -H "Content-Type: application/json" \
  -d '{"origin_region": "West US", "max_latency": 50}'
```

### Docker deployment

```bash
docker build -t region-filtering-server .
docker run -p 8080:8080 region-filtering-server
```

### Kubernetes deployment

```bash
# Point your shell to minikube's docker-daemon
# this step may vary depending on your setup
eval $(minikube docker-env)

# Check the current context
docker ps
docker images

# Build the image
docker build -t region-filtering-server:latest .

# Apply the deployment and service
kubectl apply -f server-deployment.yaml
kubectl apply -f server-service.yaml

# Check the deployment, pods, and services
kubectl get deployments
kubectl get pods
kubectl get services

# Check the service
kubectl get svc region-filtering-server

# Check detailed pod information including events
kubectl describe pods -l app=region-filtering-server

# If pods aren't appearing or are in error state, check events:
kubectl get events --sort-by='.lastTimestamp'

# Check the pod logs
kubectl logs -l app=region-filtering-server
kubectl logs deploy/region-filtering-server
kubectl logs -f $(kubectl get pods -l app=region-filtering-server -o name)

# Test the service with a test client and curl
kubectl run --rm -it --image=alpine/curl:latest test-client -- /bin/sh

curl -X POST http://region-filtering-server:8080/regions/eligible \
  -H "Content-Type: application/json" \
  -d '{"origin_region": "Italy North", "max_latency": 50}'

# Get the pod IP
kubectl get endpoints ai-inference-server
# alternative
kubectl get pods -o wide

# Get the service IP
kubectl get svc region-filtering-server

# Clean up
kubectl delete deploy/region-filtering-server
kubectl delete service/region-filtering-server
docker rmi region-filtering-server:latest
```

## TODO

- folder structure organization
- multi stage build in Dockerfile
- helm chart