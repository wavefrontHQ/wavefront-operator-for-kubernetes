package metric

import "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

func Common(clusterName string, ms []Metric) []Metric {
	transformed := ms[:0]
	for _, m := range ms {
		if m.Tags == nil {
			m.Tags = make(map[string]string, 1)
		}
		m.Tags["cluster"] = clusterName
		m.Source = clusterName
		m.Tags = TruncateTags(util.MaxTagLength, m.Tags)
		transformed = append(transformed, m)
	}
	return transformed
}
