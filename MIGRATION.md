# Migration

## Migrate from Helm Installtion

The following table lists the mapping of configurable parameters of the Wavefront Helm chart to Wavefront Operator custom resource.
Refer `config/crd/bases/wavefront.com_wavefronts.yaml` for information on the custom resource fields.

If you have collector configuration with parameters not covered below,
use `dataCollection.metrics.customConfig` to specify the name of a collector configmap in your cluster.

| Helm collector parameter            | Wavefront operator custom resource`spec`.                                               |
|-------------------------------------|------------------------------------------------------------------------------------------------------|
| `clusterName`                       | `clusterName`                                                                                        |
| `wavefront.url`	                  | `wavefrontUrl`                                                                                       |
| `wavefront.token`	                  | `wavefrontTokenSecret`                                                                               |
| `collector.enabled`	              | `dataCollection.metrics.enable`                                                                      |
| `collector.interval`	              | `dataCollection.metrics.defaultCollectionInterval`                                                   |
| `collector.useProxy`	              | `dataExport.externalWavefrontProxy`                                                                  |
| `collector.proxyAddress`	          | `dataExport.externalWavefrontProxy.Url`                                                              |
| `collector.filters`	              | `dataCollection.metrics.filters`                                                                     |
| `collector.discovery.enabled`	      | `dataCollection.metrics.enableDiscovery`                                                             |
| `collector.resources`	              | `dataCollection.metrics.nodeCollector.resources` `dataCollection.metrics.clusterCollector.resources` |
| `proxy.enabled`	                  | `dataExport.wavefrontProxy.enable`                                                                   |
| `proxy.port`	                      | `dataExport.wavefrontProxy.metricPort`                                                               |
| `proxy.httpProxyHost`	              | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| `proxy.httpProxyPort`	              | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| `proxy.useHttpProxyCAcert`	      | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| `proxy.httpProxyUser`	              | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| `proxy.httpProxyPassword`	          | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| `proxy.tracePort`	                  | `dataExport.wavefrontProxy.tracing.wavefront.port`                                                   |
| `proxy.jaegerPort`	              | `dataExport.wavefrontProxy.tracing.jaeger.port`                                                      |
| `proxy.traceJaegerHttpListenerPort` | `dataExport.wavefrontProxy.tracing.jaeger.httpPort`                                                  |
| `proxy.traceJaegerGrpcListenerPort` | `dataExport.wavefrontProxy.tracing.jaeger.grpcPort`                                                  |
| `proxy.zipkinPort`	              | `dataExport.wavefrontProxy.tracing.zipkin.port`                                                      |
| `proxy.traceSamplingRate`	          | `dataExport.wavefrontProxy.tracing.wavefront.samplingRate`                                           |
| `proxy.traceSamplingDuration`	      | `dataExport.wavefrontProxy.tracing.wavefront.samplingDuration`                                       |
| `proxy.traceJaegerApplicationName`  | `dataExport.wavefrontProxy.tracing.jaeger.applicationName`                                           |
| `proxy.traceZipkinApplicationName`  | `dataExport.wavefrontProxy.tracing.zipkin.applicationName`                                           |
| `proxy.histogramPort`	              | `dataExport.wavefrontProxy.histogram.port`                                                           |
| `proxy.histogramMinutePort`	      | `dataExport.wavefrontProxy.histogram.minutePort`                                                     |
| `proxy.histogramHourPort`	          | `dataExport.wavefrontProxy.histogram.hourPort`                                                       |
| `proxy.histogramDayPort`	          | `dataExport.wavefrontProxy.histogram.dayPort`                                                        |
| `proxy.deltaCounterPort`	          | `dataExport.wavefrontProxy.deltaCounterPort`                                                         |
| `proxy.args`	                      | `dataExport.wavefrontProxy.args`                                                                     |
| `proxy.preprocessor.rules.yaml`	  | `dataExport.wavefrontProxy.preprocessor`                                                             |

## Migrate from Manual Installation 

### Migrate wavefront proxy

See [wavefront-proxy.yaml](hack/migration/wavefront-proxy.yaml) to see how existing configuration
fields map to new [Custom Resource](deploy/kubernetes/samples/wavefront-advanced-proxy.yaml) fields.

Most of the proxy configurations could be set using environment variables for proxy container.
Here are the different proxy environment variables and how they map to operator config.
| Proxy Environment variables       | Wavefront operator custom resource `spec`                      |
|-----------------------------------|--------------------------------------------------------------- |
|`WAVEFRONT_URL`                    | `wavefrontUrl` Ex: https://<your_cluster>.wavefront.com             |
|`WAVEFRONT_TOKEN`                  | `wavefrontTokenSecret` Default: `wavefront-secret`, See below on how to create a wavefront secret.             |
|`WAVEFRONT_PROXY_ARGS`             | `wavefrontUrl` Ex: https://<your_cluster>.wavefront.com             |

Creating a Wavefront secret:
  - Create a secret using the token `kubectl create -n wavefront secret generic wavefront-secret --from-literal token=WAVEFRONT_TOKEN` 

For the below proxy configurations that is set in the environment variable `WAVEFRONT_PROXY_ARGS`, please set the corresponding operator config. 

| Wavefront Proxy args              | Wavefront operator custom resource `spec`                      |
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

If you are using any other proxy args, then set the below operator configuration parameter 
`dataExport.wavefrontProxy.args` 

If you need to set container resource request/limits for wavefront proxy, set `dataExport.wavefrontProxy.resources`

If you are using an external Wavefront Proxy, set `dataExport.externalWavefrontProxy.Url`

Did you change any other Kubernetes configuration in your proxy resource yaml? If so, please let us know as we probably might not support customizing those parameters yet in beta.


### Migrate wavefront collector

If you are using a custom collector config, then use the `dataCollection.metrics.customConfig` parameter to set the ConfigMap name. See [wavefront-advanced-collector.yaml](deploy/kubernetes/samples/wavefront-advanced-collector.yaml) for an example.
Setting this config map will override other collector configs specified in the operator. Here are the collector configs that operator supports natively as well.

| Wavefront operator custom resource `spec`          | Wavefront Collector custom ConfigMap    |
|----------------------------------------------------|-----------------------------------------|
|`clusterName`                                       | `clusterName`                           |
|`dataCollection.metrics.enableDiscovery`            | `enableDiscovery`                       |
|`dataCollection.metrics.defaultCollectionInterval`  | `defaultCollectionInterval`             |
|`dataCollection.metrics.filters.DenyList`           | `sinks.filters.metricDenyList`          |
|`dataCollection.metrics.filters.AllowList`          | `sinks.filters.metric.AllowList`        |

If you need to set container resource request/limits for wavefront collector, set `dataCollection.metrics.nodeCollector.resources` and `dataCollection.metrics.clusterCollector.resources`.

Did you change any other Kubernetes configuration in your collector resource yaml? If so, please let us know as we probably might not support customizing those parameters yet in beta.


