## Beta Notice

This project is in the beta phase and not ready for use on production environments.
For use on production environments,
refer to the Installation and Configuration sections of the [collector repo](https://github.com/wavefrontHQ/wavefront-collector-for-kubernetes)
for our original, more established processes.


# Overview of Wavefront Operator for Kubernetes

The Wavefront Operator for Kubernetes
supports deploying the Wavefront Collector and the Wavefront Proxy in Kubernetes.
This operator is based on [kubebuilder SDK](https://book.kubebuilder.io/).

## Quick Reference
- [Operator Installation](#installation)
- [Operator Validation](#validation)
- [Operator Configuration](#configuration)
- [Operator Upgrade](#upgrade)
- [Operator Removal](#removal)

## Use Cases

- Enhanced status reporting of the Kubernetes Integration to ensure that users can be proactive in ensuring their cluster and Kubernetes resources are reporting data.
- Leveraging Kubernetes Operator features to provide a more declarative mechanism for how the wavefront collector and proxy should be deployed in a Kubernetes Environment.
- Centralizing the configuration of the integration for simpler configuration of the collector and proxy.
- Providing enhanced configuration validation to reduce configuration errors and surface what needs to be corrected in order to deploy successfully.
- Enabling efficient Kubernetes resource usage by being able to scale out the cluster (leader) node and worker nodes independently.
- Providing a unified installation mechanism and form factor across VMware Tanzu product lines to ensure that users have a consistent deployment and configuration experience when deploying the Kubernetes collector and proxy.

**Note:** the collector deployed by the Operator is still a full-feature Wavefront Integration.
This list documents how the Operator extends the integration
with the goal of providing a better user experience.
For example, Istio and MySQL metrics, Telegraf configuration, etc.
are still supported.

## Architecture

![Wavefront Operator for Kubernetes Architecture](architecture.png)

# Installation

## Prerequisites

Your prerequisites will depend on your installation type.
- Manual installation uses [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm](https://helm.sh/docs/intro/install/) installation

## Deploy the Wavefront Collector and Proxy with the Operator
1. Install the Wavefront Operator

    ###### Helm 3
    ```
    helm repo add wavefront-v2beta https://projects.registry.vmware.com/chartrepo/tanzu_observability
    helm repo update
   
    kubectl create namespace wavefront
    
    helm install wavefront-v2beta wavefront-v2beta/wavefront-v2beta --namespace wavefront
    ```
   --or--
    ###### Manual
    ```
    kubectl apply -f https://github.com/wavefrontHQ/wavefront-operator-for-kubernetes/blob/main/deploy/kubernetes/wavefront-operator.yaml
    ```
2. Create a Kubernetes secret with your Wavefront Token
    ```
    kubectl create -n wavefront secret generic wavefront-secret --from-literal token=YOUR_WAVEFRONT_TOKEN
    ```
3. Create a file with your Wavefront deployment configuration.  The simplest configuration is:
    ```yaml 
    # Need to change YOUR_CLUSTER_NAME, YOUR_WAVEFRONT_URL accordingly
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
4. Deploy the Wavefront Collector and Proxy with the above configuration
    ```
    kubectl apply -f /path/to/your/wavefront.yaml
    ```
See [Configuration](#configuration) section below to learn about additional Custom Resource Configuration.

**Note**: For migrating from existing helm chart or manual deploy, see [Migration](docs/migration.md) for more information.

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


# Configuration

The Wavefront Operator is configured via a custom resource. When the resource is updated, the managing process (the operator) will pick up the changes and update the integration deployment accordingly. To update the custom resource, change the option you want in the the wavefront custom resource file and run `kubectl apply -f <your config file>.yaml`. See below for configuration options.

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

##### Note on Configuration Precedence

If you include a redundant configuration of a proxy arg
in both the dedicated Custom Resource field
and via the command line argument input in `dataExport.wavefrontProxy.args`,
the dedicated Custom Resource field will take precedence.

For example, if you specify `--histogramDistListenerPorts 40123` in `dataExport.wavefrontProxy.args`
and `dataExport.wavefrontProxy.histogram.port: 40000`,
`dataExport.wavefrontProxy.histogram.port: 40000` will take precedence.

## HTTP Proxy option

If you want more granular control over collector and proxy configuration, use the advanced configuration option, download the [wavefront-with-http-proxy.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/samples/wavefront-with-http-proxy.yaml) file.

Edit the wavefront-advanced-proxy.yaml replacing `YOUR_CLUSTER`, `YOUR_WAVEFRONT_URL`, `YOUR_HTTP_PROXY_URL` and `YOUR_HTTP_PROXY_CA_CERTIFICATE` along with any detailed configuration changes you'd like to make.

```
kubectl create -f wavefront-with-http-proxy.yaml
```


# Upgrade

The Operator installation process is idempotent,
so these commands should look familiar
to what you did during installation.

###### Helm

```
helm upgrade wavefront-v2beta wavefront-v2beta/wavefront-v2beta --namespace wavefront
```

###### Manual

Download the updated `wavefront-operator.yaml` and run the installation command below.
You can keep the secret the same or go back to the [manual installation instructions](#manual)
to create another one.
```shell
kubectl apply -f wavefront-operator-dir/wavefront-operator.yaml
```

# Removal

###### Helm

```
helm uninstall wavefront-v2beta
```

###### Manual

To undeploy the Wavefront Operator for Kubernetes, run the following command.
```
kubectl delete -f wavefront-operator.yaml
```

# Contribution 

See the [Contribution page](docs/contribution.md)
