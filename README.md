# Wavefront Operator for Kubernetes

The Wavefront Operator for Kubernetes
supports deploying the Wavefront Collector and the Wavefront Proxy in Kubernetes.
This operator is based on [kubebuilder SDK](https://book.kubebuilder.io/).

# Manual Deploy
Create a directory named wavefront-operator-dir and download the [wavefront-operator.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/wavefront-operator.yaml)
to that directory.

```
kubectl create -f kubernetes.yaml
```
Create a wavefront secret by providing YOUR_WAVEFRONT_TOKEN
```
kubectl create -n wavefront secret generic wavefront-secret --from-literal token=YOUR_WAVEFRONT_TOKEN
```

Choose between default or advanced deployment options.  

### Default option
If you're just getting started and want to advantage our experienced based default configuration, download the
[wavefront-basic.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-basic.yaml) file.


Edit the wavefront-basic.yaml replacing YOUR_CLUSTER and YOUR_WAVEFRONT_URL accordingly.

```
kubectl create -f wavefront-basic.yaml
```

### Advanced Collector option

If you want more granular control over collector and proxy configuration use the advanced configuration option, download the [wavefront-advance-collector.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-advanced-collector.yaml) file.

Edit the wavefront-advanced-collector.yaml replacing YOUR_CLUSTER and YOUR_WAVEFRONT_URL along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-advanced-collector.yaml
```

### Advanced Proxy option

If you want more granular control over collector and proxy configuration use the advanced configuration option, download the [wavefront-advance-proxy.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-advanced-proxy.yaml) file.

Edit the wavefront-advanced-proxy.yaml replacing YOUR_CLUSTER and YOUR_WAVEFRONT_URL along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-advanced-proxy.yaml
```

### HTTP Proxy option

If you want more granular control over collector and proxy configuration use the advanced configuration option, download the [wavefront-with-httpproxy.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-with-httpproxy.yaml) file.

Edit the wavefront-advanced-proxy.yaml replacing YOUR_CLUSTER, YOUR_WAVEFRONT_URL, YOUR_HTTP_PROXY_URL and YOUR_HTTP_PROXY_CA_CERTIFICATE along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-with-httpproxy.yaml
```

# Release new version of the manual deploy
Increment the version number before building
```
 PREFIX=projects.registry.vmware.com/tanzu_observability DOCKER_IMAGE=kubernetes-operator VERSION=0.10.0-alpha-5 make docker-xplatform-build generate-kubernetes-yaml
```
# Build and install locally

See the below steps to build and deploy the operator on your local kind cluster.

(Optional) Recreate your kind cluster **conveniently** from within this current repo.
You're welcome!
```
make nuke-kind
```
Run integration test
```
make integration-test 
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
# Create new local kind cluster
make nuke-kind
# Build and Deploy local operator image
make deploy-kind
# Deploy Proxy
kubectl apply -f deploy/kubernetes/samples/wavefront-basic.yaml
```


# Contributing

This is a work in progress repository.
Currently, active contribution is not supported.
