package test_helper

import (
	"bytes"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ExpectAnyMetricClient struct {
	metricSent bool
}

func NewExpectAnyMetricClient() *ExpectAnyMetricClient {
	return &ExpectAnyMetricClient{}
}

func (e *ExpectAnyMetricClient) SendMetric(_ string, _ float64, _ int64, _ string, _ map[string]string) error {
	e.metricSent = true
	return nil
}

func (e *ExpectAnyMetricClient) Flush() error {
	return nil
}

func (e *ExpectAnyMetricClient) Close() {
}

func (e *ExpectAnyMetricClient) Verify(t *testing.T) {
	t.Helper()
	assert.True(t, e.metricSent, "expected metrics to be sent")
}

type ExpectMetricClient struct {
	expectedMetricLine string

	actualMetricLines []string
}

func NewExpectedMetricClient(metricLine string) *ExpectMetricClient {
	return &ExpectMetricClient{
		expectedMetricLine: metricLine,
	}
}

func (e *ExpectMetricClient) SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error {
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

func (e *ExpectMetricClient) Verify(t *testing.T) {
	t.Helper()
	assert.Contains(t, e.actualMetricLines, e.expectedMetricLine)
}

func (e ExpectMetricClient) Flush() error {
	return nil
}

func (e ExpectMetricClient) Close() {}

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
