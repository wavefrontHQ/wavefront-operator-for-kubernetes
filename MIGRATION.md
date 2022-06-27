# Migration

## Migrate from Helm Installation

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

### Wavefront Proxy Configuration

#### References:
* See [wavefront-proxy.yaml](hack/migration/wavefront-proxy.yaml) for an example manual proxy configuration.
* See [custom-resource.yaml](deploy/kubernetes/samples/wavefront-advanced-proxy.yaml) for an example Custome Resource configuration.
* Create wavefront secret: `kubectl create -n wavefront secret generic wavefront-secret --from-literal token=WAVEFRONT_TOKEN`

Most of the proxy configurations could be set using environment variables for proxy container.
Here are the different proxy environment variables and how they map to operator config.

| Proxy Environment variables       | Wavefront operator custom resource `spec`                      |
|-----------------------------------|--------------------------------------------------------------- |
|`WAVEFRONT_URL`                    | `wavefrontUrl` Ex: https://<your_cluster>.wavefront.com             |
|`WAVEFRONT_TOKEN`                  | `wavefrontTokenSecret` Default: `wavefront-secret`. See references above for creating wavefront secret.             |
|`WAVEFRONT_PROXY_ARGS`             | `dataExport.wavefrontProxy.*` Refer to the below table for details.

Below are the proxy arguments that can be set using `WAVEFRONT_PROXY_ARGS`, which are also supported natively in the Custom Resource. 

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

Here are other custom resource configuration we support for the proxy:
* `dataExport.wavefrontProxy.args`: Used to set any other valid proxy arguments under WAVEFRONT_PROXY_ARGS which are not mentioned in the above table. 
* `dataExport.wavefrontProxy.resources`: Used to set container resource request/limits for wavefront proxy.
* `dataExport.externalWavefrontProxy.Url`: Used to set an external Wavefront Proxy.

### Wavefront Collector Configuration

If you are using a custom collector config, then use the `dataCollection.metrics.customConfig` parameter to set the ConfigMap name. See [wavefront-advanced-collector.yaml](deploy/kubernetes/samples/wavefront-advanced-collector.yaml) for an example.
Setting this config map will override other collector configs specified in the operator. Here are the collector configs that operator supports natively as well.

| Wavefront operator custom resource `spec`          | Wavefront Collector custom ConfigMap    |
|----------------------------------------------------|-----------------------------------------|
|`clusterName`                                       | `clusterName`                           |
|`dataCollection.metrics.enableDiscovery`            | `enableDiscovery`                       |
|`dataCollection.metrics.defaultCollectionInterval`  | `defaultCollectionInterval`             |
|`dataCollection.metrics.filters.DenyList`           | `sinks.filters.metricDenyList`          |
|`dataCollection.metrics.filters.AllowList`          | `sinks.filters.metric.AllowList`        |

To set container resource request/limits for wavefront collector, set `dataCollection.metrics.nodeCollector.resources` and `dataCollection.metrics.clusterCollector.resources`.


### Future Support

For configuration that has not yet been supported for legacy installation methods, please contact us for extending support after version beta.