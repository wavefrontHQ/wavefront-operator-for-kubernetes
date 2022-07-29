## Beta Notice

This project is in the beta phase and not ready for use on production environments.
For use on production environments,
see the Installation and Configuration sections of the [collector repo](https://github.com/wavefrontHQ/wavefront-collector-for-kubernetes)
for our original, more established processes.


# Overview of Wavefront Operator for Kubernetes

The Wavefront Operator for Kubernetes
supports deploying the Wavefront Collector and the Wavefront Proxy in Kubernetes.
This operator is based on [kubebuilder SDK](https://book.kubebuilder.io/).

## Quick Reference
- [Operator Installation](#installation)
- [Operator Validation](#component-status-validation)
- [Operator Configuration](#configuration)
- [Operator Upgrade](#upgrade)
- [Operator Removal](#removal)

## Why use the Wavefront Operator for Kubernetes?

The operator simplifies operational aspects of managing the Wavefront Integration. Here are some examples, with more to come!
 - Enhanced status reporting of the Wavefront Integration so that users can ensure their cluster and Kubernetes resources are reporting data.
 - Kubernetes Operator features provide a declarative mechanism for deploying the Wavefront Collector and proxy in a Kubernetes environment.
 - Centralized configuration.
 - Enhanced configuration validation to surface what needs to be corrected in order to deploy successfully.
 - Efficient Kubernetes resource usage supports scaling  out the cluster (leader) node and worker nodes independently.
 - Unified installation mechanism and form factor across VMware Tanzu product lines.

**Note:** The Collector that is deployed by the Operator still supports configuration via configmap.
For example, Istio and MySQL metrics, Telegraf configuration, etc. are still supported.

## Architecture

![Wavefront Operator for Kubernetes Architecture](architecture.png)

# Installation

## Prerequisites

The following tools are required for installing the integration.
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm 3](https://helm.sh/docs/intro/install/) (if you choose the helm installation option)

## Deploy the Wavefront Collector and Proxy with the Operator
1. Install the Wavefront Operator

   Note: Today the operator only supports deployment under the wavefront namespace.
   If you already have Wavefront deployments in that namespace, uninstall them before you install the operator.

   ```
   helm repo add wavefront-v2beta https://projects.registry.vmware.com/chartrepo/tanzu_observability
   helm repo update

   kubectl create namespace wavefront

   helm install wavefront-v2beta wavefront-v2beta/wavefront-v2beta --namespace wavefront
   ```

2. Create a Kubernetes secret with your Wavefront Token.
   See [Managing API Tokens](https://docs.wavefront.com/wavefront_api.html#managing-api-tokens) page.
   ```
   kubectl create -n wavefront secret generic wavefront-secret --from-literal token=YOUR_WAVEFRONT_TOKEN
   ```
3. Create a `wavefront.yaml` file with your Wavefront Custom Resource configuration.  The simplest configuration is:
   ```yaml
   # Need to change YOUR_CLUSTER_NAME and YOUR_WAVEFRONT_URL
   apiVersion: wavefront.com/v1alpha1
   kind: Wavefront
   metadata:
     name: wavefront
     namespace: wavefront
   spec:
     clusterName: YOUR_CLUSTER_NAME
     wavefrontUrl: YOUR_WAVEFRONT_URL
     dataCollection:
       metrics:
         enable: true
     dataExport:
       wavefrontProxy:
         enable: true
   ```
   See the [Configuration](#configuration) section below about Custom Resource Configuration.


4. Deploy the Wavefront Collector and Proxy with the above configuration
   ```
   kubectl apply -f <path_to_your_wavefront.yaml>
   ```
5. To get status for the Wavefront Integration, run the following command:
   ```
   kubectl get wavefront -n wavefront
   ```
   The command should return the following table displaying Operator instance health:
   ```
   NAME         HEALTHY      WAVEFRONT PROXY     CLUSTER COLLECTOR      NODE COLLECTOR       AGE
   wavefront      true          Running(1/1)        Running (1/1)        Running (3/3)      19h
   ```

**Note**: For details on migrating from existing helm chart or manual deploy,
see [Migration](docs/migration.md).

# Configuration

You configure the Wavefront Operator with a custom resource file.

When you update the resource file,
the Operator picks up the changes and updates the integration deployment accordingly.

To update the custom resource file:
- Open the custom resource file for edit.
- Change one or more options and save the file.
- Run `kubectl apply -f <path_to_your_config_file.yaml>`.

See below for configuration options.

We have templates for common scenarios. See the comments in each file for usage instructions.

 * [Using an existing collector ConfigMap](./deploy/kubernetes/scenarios/wavefront-collector-existing-configmap.yaml)
 * [With plugin configuration in a secret](./deploy/kubernetes/scenarios/wavefront-collector-with-plugin-secret.yaml)
 * [Filtering metrics upon collection](./deploy/kubernetes/scenarios/wavefront-collector-filtering.yaml)
 * [Defining Kubernetes resource limits](./deploy/kubernetes/scenarios/wavefront-pod-resources.yaml)
 * [Defining proxy pre-processor rules](./deploy/kubernetes/scenarios/wavefront-proxy-preprocessor-rules.yaml)
 * [Enabling proxy histogram support](./deploy/kubernetes/scenarios/wavefront-proxy-histogram.yaml)
 * [Enabling proxy tracing support](./deploy/kubernetes/scenarios/wavefront-proxy-tracing.yaml)
 * [Using an HTTP Proxy](./deploy/kubernetes/scenarios/wavefront-proxy-with-http-proxy.yaml)


You can see all configuration options in the [wavefront-full-config.yaml](./deploy/kubernetes/scenarios/wavefront-full-config.yaml).

# Upgrade

Upgrade the Wavefront Operator (both Collector and Proxy) to a new version by running the following command :

```
helm upgrade wavefront-v2beta wavefront-v2beta/wavefront-v2beta --namespace wavefront
```

Note: This command will not upgrade any existing wavefront/wavefront helm installation. See [migration.md](./docs/migration.md) for migration instructions.

# Removal

To remove the Wavefront Integration from your environment, run the following commands:

```
helm uninstall wavefront-v2beta -n wavefront
kubectl delete namespace wavefront
```

# Contribution

See the [Contribution page](docs/contribution.md)
