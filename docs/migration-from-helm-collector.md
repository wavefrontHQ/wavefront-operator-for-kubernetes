The following table lists the mapping of configurable parameters of the Wavefront Helm chart to Wavefront Operator custom resource. Refer `config/crd/bases/wavefront.com_wavefronts.yaml` for information on the custom resource fields. 

For helm collector parameters that are not mentioned below, please use `dataCollection.metrics.customConfig` configMap to configure them instead. 

| Helm collector parameter              | Wavefront operator custom resource field under `spec`.                                               |
|---------------------------------------|------------------------------------------------------------------------------------------------------|
| `clusterName`                         | `clusterName`                                                                                        |
| `wavefront.url`	                      | `wavefrontUrl`                                                                                       |
| `wavefront.token`	                    | `wavefrontTokenSecret`                                                                               |
| `collector.enabled`	                  | `dataCollection.metrics.enable`                                                                      |
| `collector.interval`	                 | `dataCollection.metrics.defaultCollectionInterval`                                                   |
| `collector.useProxy`	                 | `dataExport.externalWavefrontProxy`                                                                  |
| `collector.proxyAddress`	             | `dataExport.externalWavefrontProxy.Url`                                                              |
| `collector.filters`	                  | `dataCollection.metrics.filters`                                                                     |
| `collector.discovery.enabled`	        | `dataCollection.metrics.enableDiscovery`                                                             |
| `collector.resources`	                | `dataCollection.metrics.nodeCollector.resources` `dataCollection.metrics.clusterCollector.resources` |
| 	`proxy.enabled`	                     | `dataExport.wavefrontProxy.enable`                                                                   |
| 	`proxy.port`	                        | `dataExport.wavefrontProxy.metricPort`                                                               |
| 	`proxy.httpProxyHost`	               | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| 	`proxy.httpProxyPort`	               | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| 	`proxy.useHttpProxyCAcert`	          | `dataExport.wavefrontProxy.httpProxy.secret`                                                         | 
| 	`proxy.httpProxyUser`	               | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| 	`proxy.httpProxyPassword`	           | `dataExport.wavefrontProxy.httpProxy.secret`                                                         |
| 	`proxy.tracePort`	                   | `dataExport.wavefrontProxy.tracing.wavefront.port`                                                   | 
| 	`proxy.jaegerPort`	                  | `dataExport.wavefrontProxy.tracing.jaeger.port`                                                      | 	
| 	`proxy.traceJaegerHttpListenerPort`	 | `dataExport.wavefrontProxy.tracing.jaeger.httpPort`                                                  | 	
| 	`proxy.traceJaegerGrpcListenerPort`	 | `dataExport.wavefrontProxy.tracing.jaeger.grpcPort`                                                  | 	
| 	`proxy.zipkinPort`	                  | `dataExport.wavefrontProxy.tracing.zipkin.port`                                                      | 	
| 	`proxy.traceSamplingRate`	           | `dataExport.wavefrontProxy.tracing.wavefront.samplingRate`                                           | 
| 	`proxy.traceSamplingDuration`	       | `dataExport.wavefrontProxy.tracing.wavefront.samplingDuration`                                       | 
| 	`proxy.traceJaegerApplicationName`	  | `	dataExport.wavefrontProxy.tracing.jaeger.applicationName`                                          |
| 	`proxy.traceZipkinApplicationName`	  | `dataExport.wavefrontProxy.tracing.zipkin.applicationName`                                           | 	
| 	`proxy.histogramPort`	               | `dataExport.wavefrontProxy.histogram.port`                                                           | 
| 	`proxy.histogramMinutePort`	         | `dataExport.wavefrontProxy.histogram.minutePort`                                                     | 	
| 	`proxy.histogramHourPort`	           | `dataExport.wavefrontProxy.histogram.hourPort`                                                       | 
| 	`proxy.histogramDayPort`	            | `dataExport.wavefrontProxy.histogram.dayPort`                                                        |
| 	`proxy.deltaCounterPort`	            | `dataExport.wavefrontProxy.deltaCounterPort`                                                         |
| 	`proxy.args`	                        | `dataExport.wavefrontProxy.args`                                                                     |
| 	`proxy.preprocessor.rules.yaml`	     | `dataExport.wavefrontProxy.preprocessor`                                                             | 	
