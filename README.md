# Overview of Wavefront Operator for Kubernetes

The Wavefront Operator for Kubernetes
supports deploying the Wavefront Collector and the Wavefront Proxy in Kubernetes.
This operator is based on [kubebuilder SDK](https://book.kubebuilder.io/).

## Quick Reference

TODO link to Removal section

TODO link to Validation section

TODO link upgrade

## Beta Notice

This project is in the beta phase and not ready for usage on production environments.

## Use Cases

- Enhanced status reporting of the Kubernetes Integration to ensure that users can be proactive in ensuring their cluster and Kubernetes resources are reporting data.
- Leveraging Kubernetes Operator features to provide a more declarative mechanism for how the wavefront collector and proxy should be deployed in a Kubernetes Environment.
- Abstracting and centralizing the configuration of both the collector and proxy to enable more efficient advanced configuration of the collector and proxy.
- Providing enhanced configuration validation to reduce configuration errors and surface what needs to be corrected in order to deploy successfully.
- Enabling efficient Kubernetes resource usage by being able to scale out the cluster (leader) node and worker nodes independently.
- Providing a unified installation mechanism and form factor across VMware Tanzu product lines to ensure that users have a consistent deployment and configuration experience when deploying the Kubernetes collector and proxy.

## Architecture

![Wavefront Operator for Kubernetes Architecture](architecture.png)

# Deployment

## Prerequisites

Your prerequisites will depend on your installation type.
- Manual installation: [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Helm installation: [helm](https://helm.sh/docs/intro/install/)

## Quick Start Using Helm 3

Create a namespace to install this chart
```
kubectl create namespace wavefront
```

Create a wavefront secret by providing `YOUR_WAVEFRONT_TOKEN`
```
kubectl create -n wavefront secret generic wavefront-secret --from-literal token=YOUR_WAVEFRONT_TOKEN
```

Add chart repo by running
```
helm repo add wavefront-v2beta https://projects.registry.vmware.com/chartrepo/tanzu_observability
helm repo update
```

Install Wavefront Operator for Kubernetes
```
helm install wavefront-v2beta wavefront-v2beta/wavefront-v2beta --namespace wavefront
```

See [Configuration]() section below to learn about Custom Resource Configuration.
TODO link

## Manual

Create a directory named wavefront-operator-dir and download the [wavefront-operator.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/wavefront-operator.yaml)
to that directory.
```
kubectl apply -f wavefront-operator-dir/wavefront-operator.yaml
```

Create a wavefront secret by providing `YOUR_WAVEFRONT_TOKEN`
```
kubectl create -n wavefront secret generic wavefront-secret --from-literal token=YOUR_WAVEFRONT_TOKEN
```

Choose between default or advanced deployment options.

# Configuration

The Wavefront Operator is deployed as a Custom Resource Definition and managing process,
which allows the user to custom configure and `kubectl apply -f` a Custom Resource yaml file
to the cluster to create an instance of the Operator. See below for configuration options.

## Default option

If you're just getting started and want to take advantage of our default configurations, download the [wavefront-basic.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-basic.yaml) file.

Edit the wavefront-basic.yaml replacing `YOUR_CLUSTER` and `YOUR_WAVEFRONT_URL` accordingly.

```
kubectl create -f wavefront-basic.yaml
```

## Advanced Collector option

If you want more granular control over collector and proxy configuration, use the advanced configuration option, download the [wavefront-advanced-default-config.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-advanced-default-config.yaml) file.

Edit the wavefront-advanced-default-config.yaml replacing `YOUR_CLUSTER` and `YOUR_WAVEFRONT_URL` along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-advanced-default-config.yaml
```

## Advanced Collector with Customer defined Collector configMap option

If you want more granular control over collector and proxy configuration use the advanced configuration option, download the [wavefront-advanced-collector.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-advanced-collector.yaml) file.

Edit the wavefront-advanced-collector.yaml replacing `YOUR_CLUSTER` and `YOUR_WAVEFRONT_URL` along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-advanced-collector.yaml
```

## Advanced Proxy option

If you want more granular control over collector and proxy configuration, use the advanced configuration option, download the [wavefront-advanced-proxy.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-advanced-proxy.yaml) file.

Edit the wavefront-advanced-proxy.yaml replacing `YOUR_CLUSTER` and `YOUR_WAVEFRONT_URL` along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-advanced-proxy.yaml
```

## HTTP Proxy option

If you want more granular control over collector and proxy configuration, use the advanced configuration option, download the [wavefront-with-http-proxy.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-with-http-proxy.yaml) file.

Edit the wavefront-advanced-proxy.yaml replacing `YOUR_CLUSTER`, `YOUR_WAVEFRONT_URL`, `YOUR_HTTP_PROXY_URL` and `YOUR_HTTP_PROXY_CA_CERTIFICATE` along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-with-http-proxy.yaml
```

# Validation

## Collector and Proxy Status

To get collector and proxy status from the command line, run the following command.
```
kubectl get wavefront -n wavefront
```

It should return the following table displaying Operator instance health:
```
NAME         HEALTHY      WAVEFRONT PROXY     CLUSTER COLLECTOR      NODE COLLECTOR       AGE
wavefront      true          Running(1/1)        Running (1/1)        Running (3/3)      19h
```

# Upgrade

The Operator installation process is idempotent,
so these commands should look familiar
to what you did during installation.

## Helm

```
helm upgrade wavefront-v2beta wavefront-v2beta/wavefront-v2beta --namespace wavefront
```

## Manual

TODO link
Download the updated `wavefront-operator.yaml` and run the installation command below.
You can keep the secret the same or go back to the [manual installation instructions]()
to create another one.
```shell
kubectl apply -f wavefront-operator-dir/wavefront-operator.yaml
```

# Removal

## Helm

```
helm uninstall wavefront-v2beta
```

## Manual

To undeploy the Wavefront Operator for Kubernetes, run the following command.
```
kubectl delete -f wavefront-operator.yaml
```

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
