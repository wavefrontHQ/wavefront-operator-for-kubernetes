package util

import "os"

const OperatorName = "wavefront-controller-manager"
const ProxyName = "wavefront-proxy"
const ClusterCollectorName = "wavefront-cluster-collector"
const NodeCollectorName = "wavefront-node-collector"
const LoggingName = "wavefront-logging"
const Deployment = "Deployment"
const DaemonSet = "DaemonSet"

const MaxTagLength = 255

func Namespace() string {
	namespace, present := os.LookupEnv("NAMESPACE")
	if !present {
		return "observability-system"
	}

	return namespace
}
