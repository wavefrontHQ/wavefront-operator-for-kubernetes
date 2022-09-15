package testhelper

import (
	"bytes"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertAnyLines(t *testing.T, actualLines []string) {
	assert.NotEmpty(t, actualLines)
}

func AssertContainsLine(expectedLine string) func(t *testing.T, actualLines []string) {
	return func(t *testing.T, actualLines []string) {
		assert.Contains(t, actualLines, expectedLine)
	}
}

type MockMetricClient struct {
	assert func(t *testing.T, actualLines []string)

	actualMetricLines []string
}

func NewMockMetricClient(assert func(t *testing.T, actualLines []string)) *MockMetricClient {
	return &MockMetricClient{assert: assert}
}

func (e *MockMetricClient) SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error {
	e.actualMetricLines = append(e.actualMetricLines, metricLine(name, value, source, tags))
	return nil
}

func metricLine(name string, value float64, source string, tags map[string]string) string {
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, "%s %f source=\"%s\"", name, value, source)
	var tagNames []string
	for name := range tags {
		tagNames = append(tagNames, name)
	}
	sort.Strings(tagNames)
	for _, name := range tagNames {
		_, _ = fmt.Fprintf(buf, " %s=\"%s\"", name, tags[name])
	}
	return buf.String()
}

func (e *MockMetricClient) Verify(t *testing.T) {
	t.Helper()
	e.assert(t, e.actualMetricLines)
}

func (e MockMetricClient) Flush() error {
	return nil
}

func (e MockMetricClient) Close() {}

type StubMetricClient struct {
	SendMetricError error
	FlushError      error
}

func (s StubMetricClient) SendMetric(_ string, _ float64, _ int64, _ string, _ map[string]string) error {
	return s.SendMetricError
}

func (s StubMetricClient) Flush() error {
	return s.FlushError
}

func (s StubMetricClient) Close() {}
