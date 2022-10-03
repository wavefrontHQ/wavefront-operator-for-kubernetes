package version

import (
	"errors"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
	"regexp"
	"strconv"
)

var InvalidVersion = errors.New("invalid version (must be in semantic version format)")
var MinorVersionTooLarge = errors.New("minor version is too large (must be less than 100)")
var PatchVersionTooLarge = errors.New("patch version is too large (must be less than 100)")

// semverRegex is taken from https://semver.org
var semverRegex = regexp.MustCompile("^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$")

type Sender struct {
	client senders.MetricClient
}

func NewSender(client senders.MetricClient) *Sender {
	return &Sender{client: client}
}

func (s *Sender) SendVersion(version string, cluster string) error {
	parts := semverRegex.FindStringSubmatch(version)
	if len(parts) == 0 {
		return InvalidVersion
	}
	major, _ := strconv.ParseFloat(parts[1], 64)
	minor, _ := strconv.ParseFloat(parts[2], 64)
	patch, _ := strconv.ParseFloat(parts[3], 64)
	if minor >= 100.0 {

		return MinorVersionTooLarge
	}
	if patch >= 100.0 {

		return PatchVersionTooLarge
	}
	return s.client.SendMetric(
		"kubernetes.observability.version",
		encodeSemver(major, minor, patch),
		0,
		cluster,
		nil,
	)
}

func encodeSemver(major float64, minor float64, patch float64) float64 {
	const versionOffset = 0.01
	return major + minor*versionOffset + patch*versionOffset*versionOffset
}
