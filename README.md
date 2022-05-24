# Wavefront Operator for Kubernetes

The Wavefront Operator for Kubernetes
supports deploying the Wavefront Collector and the Wavefront Proxy in Kubernetes.
This operator is based on [kubebuilder SDK](https://book.kubebuilder.io/).

# Installation

See the below steps to build and deploy the operator on your local kind cluster.

(Optional) Recreate your kind cluster **conveniently** from within this current repo.
You're welcome!
```
pushd ~/workspace/wavefront-collector-for-kubernetes
    make nuke-kind
popd
```

Generate the Custom Resource **Definition** (`manifests`),
and apply it to the current cluster (`install`)
(see below to create an **instance** of the Custom Resource):
```
make manifests install
```
**NOTE**: Currently Kubebuilder requires **go 1.17**. If the above step fails please verify that the go version is set to 1.17 in your environment.

Build the controller manager binary from the go code:
```
make build
```

Run the controller manager on the local cluster:
```
OPERATOR_VERSION=1
make docker-build IMG=kind-local/wavefront-operator:${OPERATOR_VERSION}
kind load docker-image kind-local/wavefront-operator:${OPERATOR_VERSION}
make deploy IMG=kind-local/wavefront-operator:${OPERATOR_VERSION}
```

or 

```
# Create new local kind cluster
make nuke-kind
# Build and Deploy local operator image
make deploy-kind
# Deploy Proxy
kubectl apply -f config/samples/_v1alpha1_wavefront.yaml 
```

Finally, create the **instance** of the **Custom Resource**,
which Kubernetes will validate against the schema in the Custom Resource **Definition**:
```
kubectl apply -f config/samples/
```

# Manual Deploy
Create a directory named wavefront-operator-dir and download the following files to that directory:
* [kubernetes.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/kubernetes.yaml)
* [wavefront-basic.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-basic.yaml)

```
kubectl apply -f kubernetes.yaml
```

Create a wavefront secret
```
kubectl create -n wavefront secret generic wavefront-secret --from-literal token=YOUR_WAVEFRONT_TOKEN
```

Edit the wavefront-basic.yaml replacing YOUR_CLUSTER and YOUR_WAVEFRONT_URL accordingly.
```
kubectl apply -f wavefront-basic.yaml
```

# Contributing

This is a work in progress repository.
Currently, active contribution is not supported.
