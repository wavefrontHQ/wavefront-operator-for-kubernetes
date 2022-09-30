package version

import (
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
	"regexp"
	"strconv"
)

var semverRegex = regexp.MustCompile("^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$")

type Sender struct {
	client senders.MetricClient
}

func NewSender(client senders.MetricClient) *Sender {
	return &Sender{client: client}
}

func (s *Sender) SendVersion(version string, cluster string) error {
	// get version from ldflags
	submatches := semverRegex.FindStringSubmatch(version)
	major, _ := strconv.ParseFloat(submatches[1], 64)
	minor, _ := strconv.ParseFloat(submatches[2], 64)
	patch, _ := strconv.ParseFloat(submatches[3], 64)

	return s.client.SendMetric("kubernetes.observability.version", major+minor/100+patch/1000, 0, cluster, nil)
}
