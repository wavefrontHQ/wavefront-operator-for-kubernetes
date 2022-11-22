package metric

type Metric struct {
	Name          string
	Value         float64
	Source        string
	Tags          map[string]string
	ComponentName string
}
