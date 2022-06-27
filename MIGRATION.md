# Migration

## Helm

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

## Manual

See [wavefront-proxy.yaml](hack/migration/wavefront-proxy.yaml) to see how existing configuration
fields map to new [Custom Resource](deploy/kubernetes/samples/wavefront-advanced-proxy.yaml) fields.

There are a few important proxy configurations that we support natively in the operator.
For the below proxy configurations that we support natively, please use the corresponding operator config. 

| Wavefront Proxy args              | Wavefront operator custom resource `spec`                      |
|-----------------------------------|--------------------------------------------------------------- |
|`--preprocessorConfigFile`         | `dataExport.wavefrontProxy.preprocessor` ConfigMap             |
|`--proxyHost`                      | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
|`--proxyPort`                      | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
|`--proxyUser`                      | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
|`--proxyPassword`                  | `dataExport.wavefrontProxy.httpProxy.secret` Secret            |
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







