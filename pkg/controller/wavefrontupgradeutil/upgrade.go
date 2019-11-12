package wavefrontupgradeutil

const (
	ProxyImageName = "proxy"

	CollectorImageName = "wavefront-kubernetes-collector"
)

func GetLatestVersion(cr string, majorVersion string) string {
	return "4.38"
}