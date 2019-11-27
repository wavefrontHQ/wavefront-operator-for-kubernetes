# Installation and Configuration of Wavefront Operator on OpenShift
This page contains the Installation and Configuration steps for full-stack monitoring of OpenShift clusters using Wavefront Operator.

**Supported Versions**: Openshift Enterprise 4.x

## Prerequisite
* Generate Wavefront API token as given in the [document](https://docs.wavefront.com/usersaccountmanaging.html#generating-an-api-token).

  **or**

* You must have a external proxy installed and configured that is reachable to the Openshift Cluster.

## Installation and Configuration of Wavefront Operator 

1.  Login into Openshift Web UI as administrator.
2.  Create a project with name "wavefront".
3.  From the Left pane navigate to the "Catalog" → "OperatorHub".
4.  From the list of Operator types select "Monitoring" → "Wavefront".
5.  Click on the Wavefront Operator and click Install.
6.  Subscribe for the Operator by selecting "wavefront" as namespace.
7.  Once the subscription is successful the operator will be listed under "Installed Operators" and it deploys Wavefront Proxy and Wavefront Collector CRD's into the project.
8.  Now deploy the proxy by navigating to Installed Operators → Wavefront Operator → Wavefront Proxy → Create New
9.  Create Wavefront Proxy Custom Resource by filling below parameters in proxy Spec and leave rest of the values as defaults.
    * token→ Wavefront Token
    * url → Wavefront cluster url
10.  Click on Create.  This will deploy proxy and service with the name "example-wavefrontproxy" and port 2878 as metric port. Also Operator creates PVC with the name "wavefront-proxy-storage" using default underlying PV.
11. Now deploy the collector by navigating to Installed Operators → Wavefront Operator → Wavefront Collector → Create New.
12. Click on create without changing any values in the proxy definition.

As default parameters are used, collector runs as daemonset and use "example-wavefrontproxy" as sink.  It collects metrics from the kubernetes api server, kube-state-metrics and auto discovers the pods and services that expose metrics and dynamically start collecting metrics for the targets.

Now login into Wavefront and search for the "openshift-demo-cluster" in kubernetes integration dashboards.

## Using External Proxy
Wavefront Collector can be configured to use external proxy using below steps:
1. Download example configuration [file](https://raw.githubusercontent.com/wavefrontHQ/wavefront-collector-for-kubernetes/master/deploy/examples/openshift-config.yaml).
2. Update `sinks.proxyAddress` with your external proxy address. Please refer this [document](https://github.com/wavefrontHQ/wavefront-collector-for-kubernetes/blob/master/docs/configuration.md) for more configuration options.
3. Create a configMap using downloaded file under the project where operator is deployed.

   Example:-
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: collector-config
     namespace: wavefront-collector
   data:
     collector.yaml: |
       clusterName: k8s-cluster
       enableDiscovery: true
       enableEvents: true
       flushInterval: 30s
       sinks:
         - proxyAddress: external-proxy:2878
       ...
       ...
   ```
4. Now deploy the collector by navigating to Installed Operators → Wavefront Operator → Wavefront Collector → Create New.
5. Set `spec.useOpenshiftDefaultConfig` to `false` and `spce.configName` to the configMap name created in step 3.
6. Click on create.


## Advanced Wavefront Proxy Configuration
1. Create a configMap under the project where the operator is deployed.

   Example:-
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: advanced-config
     namespace: wavefront
   data:
     wavefront.conf: |
       prefix = dev
       customSourceTags = <YOUR_K8S_CLUSTER>
2. Now deploy the proxy by navigating to Installed Operators → Wavefront Operator → Wavefront Proxy → Create New.
3. Set `spec.advanced` to the configMap name created in Step 1.
4. Set `spec.token` to Wavefront API token and `spec.url` to Wavefront URL.
5. Click on create.

**Note**:- Refer this [document](https://docs.wavefront.com/proxies_configuring.html#general-proxy-properties-and-examples) for more details on proxy configuration properties.

## Configuring Wavefront Proxy Preprocessor Rules

1. Create a configMap under the project where the operator is deployed.
   
   Example:-
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: preprocessor-config
     namespace: wavefront
   data:
      rules.yaml: |
        '2878':
          - rule    : add-cluster-tag
            action  : addTag
            tag     : env
            value   : dev
    ```
2. Now deploy the proxy by navigating to Installed Operators → Wavefront Operator → Wavefront Proxy → Create New.
3. Set `spec.preprocessor` to the configMap name created in Step 1.
4. Set `spec.token` to Wavefront API token and `spec.url` to Wavefront URL.
5. Click on create.

**Note**:- Refer this [document](https://docs.wavefront.com/proxies_preprocessor_rules.html#rule-configuration-file) for more details on preprocessor rules.
