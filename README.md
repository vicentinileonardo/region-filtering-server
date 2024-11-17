# Region filtering server

## Description

This repository is part of a project: **Carbon-aware workload scheduling in a multi-cloud environment**.

This is a simple server that filters cloud regions based on a **region of origin** and a **latency threshold**. 

Only **Azure regions** are supported at the moment.

## Data

Data is obtained from the Azure Docs. In particular, the data inside `latency_matrix.csv` is extracted from the following page: [Azure Network Latency](https://raw.githubusercontent.com/MicrosoftDocs/azure-docs/refs/heads/main/articles/networking/azure-network-latency.md).

Inside the `scripts` folder, there is a simple utility script that can be used to merge csv files into a single csv file representing the latency between all regions.
This can be useful to generate a new latency matrix if the original data changes in the documentation.

The file `region_city_mapping.csv` is a manually created file that maps Azure regions to cities. 
This is temporary, obtained from the following page: [Azure Regions](https://www.azurespeed.com/Information/AzureRegions), and should be replaced with direct data from Azure, if available.

## Setup

### Local deployment

```bash
go run main.go
```

Test the server by sending a request like the following:
```bash
curl -X POST http://localhost:8080/regions/eligible \
  -H "Content-Type: application/json" \
  -d '{"cloud_provider": "azure", "cloud_provider_origin_region": "Italy North", "max_latency": 15}'
```

Expected response:
```json
{
  "cloud_provider":"azure",
  "eligible_regions":
  [
    {
      "name":"Switzerland West",
      "iso_country_code_a2":"CH",
      "electricity_maps_region":"N/A",
      "physical_location":""
    },
    {
      "name":"Switzerland North",
      "iso_country_code_a2":"CH",
      "electricity_maps_region":"N/A",
      "physical_location":"Zurich"
    },
    {
      "name":"France South",
      "iso_country_code_a2":"FR",
      "electricity_maps_region":"N/A",
      "physical_location":""
    },
    {
      "name":"Italy North",
      "iso_country_code_a2":"IT",
      "electricity_maps_region":"N/A",
      "physical_location":"Milan"
    }
  ]
}
```

### Docker deployment

```bash
docker build -t region-filtering-server .
docker run -p 8080:8080 region-filtering-server
```

### Kubernetes deployment

Point your shell to minikube's docker-daemon, this step may vary depending on your setup:
```bash
eval $(minikube docker-env)
```
Check the current Docker context:
```bash
docker ps
docker images
```

Build the image:
```bash
docker build -t region-filtering-server:latest .
```

Apply the deployment and service:
```bash
kubectl apply -f server-deployment.yaml
kubectl apply -f server-service.yaml
```

Check the deployment, pods, and services:
```bash
kubectl get deployments
kubectl get pods
kubectl get services
```

Check the service:
```bash
kubectl get svc region-filtering-server
```

Check detailed pod information including events:
```bash
kubectl describe pods -l app=region-filtering-server
```

If pods aren't appearing or are in error state, check events:
```bash
kubectl get events --sort-by='.lastTimestamp'
```

Check the pod logs:
```bash
kubectl logs -l app=region-filtering-server
kubectl logs deploy/region-filtering-server
kubectl logs -f $(kubectl get pods -l app=region-filtering-server -o name)
```

Test the service with a test client and `curl`:
```bash
kubectl run --rm -it --image=alpine/curl:latest test-client -- /bin/sh

curl -X POST http://region-filtering-server:8080/regions/eligible \
  -H "Content-Type: application/json" \
  -d '{"cloud_provider": "azure", "cloud_provider_origin_region": "West US", "max_latency": 50}'
```

Expected response:
```json
{ 
  "cloud_provider":"azure",
  "eligible_regions":
  [
    {
      "name":"Central US",
      "iso_country_code_a2":"US",
      "electricity_maps_region":"N/A",
      "physical_location":"Iowa"
    },
    {
      "name":"South Central US",
      "iso_country_code_a2":"US",
      "electricity_maps_region":"N/A",
      "physical_location":"Texas"
    },
    {
      "name":"West US",
      "iso_country_code_a2":"US",
      "electricity_maps_region":"N/A",
      "physical_location":"California"
    },
    {
      "name":"North Central US",
      "iso_country_code_a2":"US",
      "electricity_maps_region":"N/A",
      "physical_location":"Illinois"
    },
    {
      "name":"West Central US",
      "iso_country_code_a2":"US",
      "electricity_maps_region":"N/A",
      "physical_location":"Wyoming"
    }
  ]
}
```

Get the pod IP (if needed for debugging purposes):
```bash
kubectl get endpoints region-filtering-server
# alternative
kubectl get pods -o wide
```

Get the service IP (if needed for debugging purposes):
```bash
kubectl get svc region-filtering-server
```

Clean up:
```bash
kubectl delete deploy/region-filtering-server
kubectl delete service/region-filtering-server
docker rmi region-filtering-server:latest
```

## TODO

- update readme
- folder structure organization
- add other cloud providers (PoC, not official data)
- test that provided origin region is a valid region for the specified cloud provider
- multi stage build in Dockerfile
- helm chart
- probably it would be useful to have a table with the mapping to Electricity Maps regions or a column in an existing table