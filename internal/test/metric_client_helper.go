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
	assert.True(t, e.metricSent, "expected metrics to be sent")
}

type ExpectMetricSender struct {
	expectedMetricLine string

	actualMetricLines []string
}

func NewExpectedMetricClient(metricLine string) *ExpectMetricSender {
	return &ExpectMetricSender{
		expectedMetricLine: metricLine,
	}
}

func (e *ExpectMetricSender) SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "%s %f source=\"%s\"", name, value, source)
	var tagNames []string
	for name := range tags {
		tagNames = append(tagNames, name)
	}
	sort.Strings(tagNames)
	for _, name := range tagNames {
		fmt.Fprintf(buf, " %s=\"%s\"", name, tags[name])
	}
	e.actualMetricLines = append(e.actualMetricLines, buf.String())
	return nil
}

func (e *ExpectMetricSender) Verify(t *testing.T) {
	t.Helper()
	assert.Contains(t, e.actualMetricLines, e.expectedMetricLine)
}

func (e ExpectMetricSender) Flush() error {
	return nil
}

func (e ExpectMetricSender) Close() {}

type StubMetricSender struct {
	SendMetricError error
	FlushError      error
}

func (s StubMetricSender) SendMetric(_ string, _ float64, _ int64, _ string, _ map[string]string) error {
	return s.SendMetricError
}

func (s StubMetricSender) Flush() error {
	return s.FlushError
}

func (s StubMetricSender) Close() {}
