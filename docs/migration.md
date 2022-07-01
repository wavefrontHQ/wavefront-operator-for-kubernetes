# Migration
This is a beta trial migration doc for the operator from collector manual and helm installation.

## Migrate from Helm Installation

The following table lists the mapping of configurable parameters of the Wavefront Helm chart to Wavefront Operator Custom Resource.

See [Custom Resource Scenarios](/deploy/kubernetes/scenarios) for examples or refer to [config/crd/bases/wavefront.com_wavefronts.yaml](../config/crd/bases/wavefront.com_wavefronts.yaml) for information on all Custom Resource fields.

| Helm collector parameter           | Wavefront operator Custom Resource `spec`.                                                           | Description                                                                                                                                                    |
|------------------------------------|------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `clusterName`                      | `clusterName`                                                                                        | ClusterName is a unique name for the Kubernetes cluster to be identified via a metric tag on Wavefront                                                         |
| `wavefront.url`                    | `wavefrontUrl`                                                                                       | Wavefront URL for your cluster. Ex: https://<your_cluster>.wavefront.com                                                                                       |
| `wavefront.token`                  | `wavefrontTokenSecret`                                                                               | WavefrontTokenSecret is the name of the secret that contains a wavefront API Token.                                                                            |
| `collector.enabled`                | `dataCollection.metrics.enable`                                                                      | Metrics holds the configuration for node and cluster collectors.                                                                                               |
| `collector.interval`               | `dataCollection.metrics.defaultCollectionInterval`                                                  | Default metrics collection interval. Defaults to 60s.                                                                                                          |
| `collector.useProxy`               | `dataExport.externalWavefrontProxy`                                                                  |                                                                                                                                                                |
| `collector.proxyAddress`           | `dataExport.externalWavefrontProxy.Url`                                                              | Url is the proxy URL that the collector sends metrics to.                                                                                                      |
| `collector.filters`                | `dataCollection.metrics.filters`                                                                     | Filters to apply towards all metrics collected by the collector.                                                                                               |
| `collector.discovery.enabled`      | `dataCollection.metrics.enableDiscovery`                                                             | Rules based and Prometheus endpoints auto-discovery. Defaults to true.                                                                                         |
| `collector.resources`              | `dataCollection.metrics.nodeCollector.resources` `dataCollection.metrics.clusterCollector.resources` | Compute resources required by the node and cluster collector containers.                                                                                       |
| `proxy.enabled`                    | `dataExport.wavefrontProxy.enable`                                                                   | Enable is whether to enable the wavefront proxy. Defaults to true.                                                                                             |
| `proxy.port`                       | `dataExport.wavefrontProxy.metricPort`                                                               | MetricPort is the port for sending Wavefront data format metrics. Defaults to 2878.                                                                            |
| `proxy.httpProxyHost`              | `dataExport.wavefrontProxy.httpProxy.secret`                                                         | Name of the secret containing the HttpProxy configuration.                                                                                                     |
| `proxy.httpProxyPort`              | `dataExport.wavefrontProxy.httpProxy.secret`                                                         | Name of the secret containing the HttpProxy configuration.                                                                                                     |
| `proxy.useHttpProxyCAcert`         | `dataExport.wavefrontProxy.httpProxy.secret`                                                         | Name of the secret containing the HttpProxy configuration.                                                                                                     |
| `proxy.httpProxyUser`              | `dataExport.wavefrontProxy.httpProxy.secret`                                                         | Name of the secret containing the HttpProxy configuration.                                                                                                     |
| `proxy.httpProxyPassword`          | `dataExport.wavefrontProxy.httpProxy.secret`                                                         | Name of the secret containing the HttpProxy configuration.                                                                                                     |
| `proxy.tracePort`                  | `dataExport.wavefrontProxy.tracing.wavefront.port`                                                  | Port for sending distributed wavefront format tracing data (usually 30000)                                                                                     |
| `proxy.jaegerPort`                 | `dataExport.wavefrontProxy.tracing.jaeger.port`                                                      | Port for Jaeger format tracing data (usually 30001)                                                                                                            |
| `proxy. traceJaegerHttpListenerPort`| `dataExport.wavefrontProxy.tracing.jaeger. httpPort`                                                 | HttpPort for Jaeger Thrift format data (usually 30080)                                                                                                         |
| `proxy. traceJaegerGrpcListenerPort`| `dataExport.wavefrontProxy.tracing.jaeger. grpcPort`                                                 | GrpcPort for Jaeger GRPC format data (usually 14250)                                                                                                           |
| `proxy.zipkinPort`                 | `dataExport.wavefrontProxy.tracing.zipkin.port`                                                      | Port for Zipkin format tracing data (usually 9411)                                                                                                             |
| `proxy.traceSamplingRate`          | `dataExport.wavefrontProxy.tracing.wavefront. samplingRate`                                          | SamplingRate Distributed tracing data sampling rate (0 to 1)                                                                                                   |
| `proxy.traceSamplingDuration`      | `dataExport.wavefrontProxy.tracing.wavefront. samplingDuration`                                      | SamplingDuration When set to greater than 0, spans that exceed this duration will force trace to be sampled (ms)                                               |
| `proxy. traceJaegerApplicationName` | `dataExport.wavefrontProxy.tracing.jaeger. applicationName`                                          | Custom application name for traces received on Jaeger's Http or Gprc port.                                                                                     |
| `proxy. traceZipkinApplicationName` | `dataExport.wavefrontProxy.tracing.zipkin. applicationName`                                          | Custom application name for traces received on Zipkin's port.                                                                                                  |
| `proxy.histogramPort`              | `dataExport.wavefrontProxy.histogram.port`                                                           | Port for wavefront histogram distributions (usually 40000)                                                                                                     |
| `proxy.histogramMinutePort`        | `dataExport.wavefrontProxy.histogram.minutePort`                                                     | Port to accumulate 1-minute based histograms on Wavefront data format (usually 40001)                                                                          |
| `proxy.histogramHourPort`          | `dataExport.wavefrontProxy.histogram.hourPort`                                                       | Port to accumulate 1-hour based histograms on Wavefront data format (usually 40002)                                                                            |
| `proxy.histogramDayPort`           | `dataExport.wavefrontProxy.histogram.dayPort`                                                        | Port to accumulate 1-day based histograms on Wavefront data format (usually 40002)                                                                             |
| `proxy.deltaCounterPort`           | `dataExport.wavefrontProxy.deltaCounterPort`                                                         | Port to send delta counters on Wavefront data format (usually 50000)                                                                                           |
| `proxy.args`                       | `dataExport.wavefrontProxy.args`                                                                     | Additional Wavefront proxy properties can be passed as command line arguments in the `--<property_name> <value>` format. Multiple properties can be specified. |
| `proxy.preprocessor.rules.yaml`    | `dataExport.wavefrontProxy.preprocessor`                                                             | Name of the configmap containing a rules.yaml key with proxy preprocessing rules                                                                               |


If you have collector configuration with parameters not covered above, please reach out to us.

## Migrate from Manual Installation 

### Wavefront Proxy Configuration

#### References:
* See [Custom Resource Scenarios](/deploy/kubernetes/scenarios) for proxy configuration examples.
* Create wavefront secret: `kubectl create -n wavefront secret generic wavefront-secret --from-literal token=WAVEFRONT_TOKEN`

Most of the proxy configurations could be set using environment variables for proxy container.
Here are the different proxy environment variables and how they map to operator config.

| Proxy Environment variables       | Wavefront operator Custom Resource `spec`                                                      |
|-----------------------------------|------------------------------------------------------------------------------------------------|
|`WAVEFRONT_URL`                    | `wavefrontUrl` Ex: https://<your_cluster>.wavefront.com                                        |
|`WAVEFRONT_TOKEN`                  | `WAVEFRONT_TOKEN` is now stored in a Kubernetes secret; see **Create wavefront secret** above. |
|`WAVEFRONT_PROXY_ARGS`             | `dataExport.wavefrontProxy.*` Refer to the below table for details.                            |

Below are the proxy arguments that are specified in `WAVEFRONT_PROXY_ARGS`, which are currently supported natively in the Custom Resource. 

| Wavefront Proxy args              | Wavefront operator Custom Resource `spec`                      |
|-----------------------------------|--------------------------------------------------------------- |
|`--preprocessorConfigFile`         | `dataExport.wavefrontProxy.preprocessor` ConfigMap             |
|`--proxyHost`                      | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
|`--proxyPort`                      | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
|`--proxyUser`                      | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
|`--proxyPassword`                  | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
|`--pushListenerPorts`              | `dataExport.wavefrontProxy.metricPort`                         |
|`--deltaCounterPorts`              | `dataExport.wavefrontProxy.deltaCounterPort`                   |
|`--traceListenerPorts`             | `dataExport.wavefrontProxy.tracing.wavefront.port`             |
|`--traceSamplingRate`              | `dataExport.wavefrontProxy.tracing.wavefront.samplingRate`     |
|`--traceSamplingDuration`          | `dataExport.wavefrontProxy.tracing.wavefront.samplingDuration` |
|`--traceZipkinListenerPorts`       | `dataExport.wavefrontProxy.tracing.zipkin.port`                |
|`--traceZipkinApplicationName`     | `dataExport.wavefrontProxy.tracing.zipkin.applicationName`     |
|`--traceJaegerListenerPorts`       | `dataExport.wavefrontProxy.tracing.jaeger.port`                |
|`--traceJaegerHttpListenerPorts`   | `dataExport.wavefrontProxy.tracing.jaeger.httpPort`            |
|`--traceJaegerGrpcListenerPorts`   | `dataExport.wavefrontProxy.tracing.jaeger.grpcPort`            |
|`--traceJaegerApplicationName`     | `dataExport.wavefrontProxy.tracing.jaeger.applicationName`     |
|`--histogramDistListenerPorts`     | `dataExport.wavefrontProxy.histogram.port`                     |
|`--histogramMinuteListenerPorts`   | `dataExport.wavefrontProxy.histogram.minutePort`               |
|`--histogramHourListenerPorts`     | `dataExport.wavefrontProxy.histogram.hourPort`                 |
|`--histogramDayListenerPorts`      | `dataExport.wavefrontProxy.histogram.dayPort`                  |

Other supported Custom Resource configuration:
* `dataExport.wavefrontProxy.args`: Used to set any `WAVEFRONT_PROXY_ARGS` configuration not mentioned in the above table. See [wavefront-proxy-args.yaml](../deploy/kubernetes/scenarios/wavefront-proxy-args.yaml) for an example.
* `dataExport.wavefrontProxy.resources`: Used to set container resource request or limits for Wavefront Proxy. See [wavefront-pod-resources.yaml](../deploy/kubernetes/scenarios/wavefront-pod-resources.yaml) for an example.
* `dataExport.externalWavefrontProxy.Url`: Used to set an external Wavefront Proxy. See [wavefront-collector-external-proxy.yaml](../deploy/kubernetes/scenarios/wavefront-collector-external-proxy.yaml) for an example.

### Wavefront Collector Configuration

Wavefront Collector `ConfigMap` changes:
* Wavefront Collector ConfigMap changed from `wavefront-collector` to `wavefront` namespace.
* `sinks.proxyAddress` changed from `wavefront-proxy.default.svc.cluster.local:2878` to `wavefront-proxy:2878`.

Custom Resource `spec` changes:
* Update Custom Resource configuration`dataCollection.metrics.customConfig` with the created ConfigMap name.
See [wavefront-collector-existing-configmap.yaml](../deploy/kubernetes/scenarios/wavefront-collector-existing-configmap.yaml) for an example.

Other supported Custom Resource configurations:
* `dataCollection.metrics.nodeCollector.resources`: Used to set container resource request or limits for Wavefront node collector.
* `dataCollection.metrics.clusterCollector.resources`: Used to set container resource request or limits for Wavefront cluster collector.
See [wavefront-pod-resources.yaml](../deploy/kubernetes/scenarios/wavefront-pod-resources.yaml) for an example.

### Future Support

If you come across something that cannot be configured with the new Operator,
please contact us.
