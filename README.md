# Wavefront Operator for Kubernetes

The Wavefront Operator for Kubernetes
supports deploying the Wavefront Collector and the Wavefront Proxy in Kubernetes.
This operator is based on [kubebuilder SDK](https://book.kubebuilder.io/).

# Installation

If you are editing the API definitions,
generate the manifests such as CRs or CRDs using:
```
make manifests
```

Install the CRDs into the cluster:
```
make install
```

To build and deploy the operator on local kind cluster follow the below steps.

```
make build
make manifests
make install
make docker-build IMG=kind-local/wavefront-operator
kind load docker-image kind-local/wavefront-operator
kubectl apply -f config/samples/
```
# Contributing

This is a work in progress repository.
Currently, active contribution is not supported.

