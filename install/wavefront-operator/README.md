# Wavefront Operator for Kubernetes

[Helm](https://helm.sh/) is a package manager for Kubernetes. You can use Helm
for installing the Wavefront Operator in your Kubernetes cluster.

## Prerequisites

To deploy this operator, you will need a cluster with the following minimum setup:

* Kubernetes v1.12.0
* Helm v2.10.0

## Installing the Chart
The required options for the chart are:
- clusterName
- wavefront.url
- wavefront.token

To install the chart with a release name `test`:

```
helm install --name test ./wavefront-operator --set wavefront.url=https://YOUR_CLUSTER.wavefront.com,wavefront.token=YOUR_API_TOKEN,clusterName=YOUR_CLUSTER_NAME --namespace test-namespace
```

### Troubleshooting:

- CRD already exists:
```
Error: customresourcedefinitions.apiextensions.k8s.io <"wavefrontcollectors.wavefront.com"> already exists
```

If you see the above error (can be seen when trying to create multiple releases), then try running 
the helm command with "--no-crd-hook" flag.

```
helm install --name test ./wavefront-operator --set wavefront.url=https://YOUR_CLUSTER.wavefront.com,wavefront.token=YOUR_API_TOKEN,clusterName=YOUR_CLUSTER_NAME --namespace test-namespace --no-crd-hook
```

## Uninstalling the Chart
To uninstall/delete a deployed chart release:
```
helm delete <release-name> --purge
```

CRDs created by this chart are not removed as part of helm delete. To remove the CRDs:
```
kubectl delete crd wavefrontcollectors.wavefront.com
kubectl delete crd wavefrontproxies.wavefront.com
```
