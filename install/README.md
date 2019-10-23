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
$ helm repo add wavefront 'https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator/master/install/'
$ helm repo update
```

### Install the Chart
The required options for the chart are:
- clusterName
- wavefront.url
- wavefront.token

To deploy a release named "test" into a namespace "test-ns":
```
$ helm install --name test wavefront/wavefront-operator --set wavefront.url=https://YOUR_CLUSTER.wavefront.com,wavefront.token=YOUR_API_TOKEN,clusterName=YOUR_CLUSTER_NAME --namespace test-ns
```

## Uninstallation
To uninstall/delete a deployed chart named "test":
```
$ helm delete test --purge
```

CRDs created by this chart are not removed as part of helm delete. To remove the CRDs:
```
kubectl delete crd wavefrontcollectors.wavefront.com
kubectl delete crd wavefrontproxies.wavefront.com
```
