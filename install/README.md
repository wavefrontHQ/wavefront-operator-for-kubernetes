# Wavefront Operator Helm Chart

[Helm](https://helm.sh/) is a package manager for Kubernetes. You can use Helm for installing the Wavefront Operator in your Kubernetes cluster.

## Introduction
This chart will deploy the Wavefront Collector for Kubernetes and the Wavefront Proxy to your Kubernetes cluster. You can use this chart to install multiple Wavefront Proxy releases, though only one Wavefront Kubernetes collector per cluster should be used.

## Prerequisites

To deploy this operator, you will need a cluster with the following minimum setup:

* Kubernetes v1.12.0 or above
* Helm v2.10.0 or above

## Installation

### Add the Wavefront Repo
```
helm repo add wavefront 'https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator/master/install/'
helm repo update
```

### Install the Chart
The required options for the chart are:
- clusterName
- wavefront.url
- wavefront.token

To deploy a release named "test" into a namespace "test-ns":
```
helm install --name test wavefront/wavefront-operator --set wavefront.url=https://YOUR_CLUSTER.wavefront.com,wavefront.token=YOUR_API_TOKEN,clusterName=YOUR_CLUSTER_NAME --namespace test-ns
```

### Troubleshooting:

- CRD already exists:
```
Error: customresourcedefinitions.apiextensions.k8s.io <"wavefrontcollectors.wavefront.com"> already exists
```

If you see the above error (can be seen when trying to create multiple releases), then try running 
the helm command with "--no-crd-hook" flag.

```
helm install --name test wavefront/wavefront-operator --set wavefront.url=https://YOUR_CLUSTER.wavefront.com,wavefront.token=YOUR_API_TOKEN,clusterName=YOUR_CLUSTER_NAME --namespace test-namespace --no-crd-hook
```

## Uninstallation
To uninstall/delete a deployed chart named "test":
```
helm delete test --purge
```

CRDs created by this chart are not removed as part of helm delete. To remove the CRDs:
```
kubectl delete crd wavefrontcollectors.wavefront.com
kubectl delete crd wavefrontproxies.wavefront.com
```

## Development
To update the helm chart:
- Update the files under `./wavefront-operator`
- Increment `version` in Chart.yaml to next desired version.
- From inside `install` directory run the below
    - Run `helm package ./wavefront-operator` to update the tgz file
    - Run `helm repo index .` to update the `index.yaml`
- Commit the changes to this repo
