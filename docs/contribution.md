# Contribution and Dev Work

## Community contribution

This repository is a work in progress.
Currently, community contribution is not supported.

## Release new version of the manual deploy

Increment the version number before building
```
 PREFIX=projects.registry.vmware.com/tanzu_observability DOCKER_IMAGE=kubernetes-operator VERSION=0.10.0-alpha-7 make docker-xplatform-build generate-kubernetes-yaml
```

## Build and install locally

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
