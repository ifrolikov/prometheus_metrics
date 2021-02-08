package prometheus_metrics

import (
	"time"
)

type DummyCollector struct {

}

func (d DummyCollector) ObserveTimer(name string, startTime time.Time, labels map[string]string) error {
	return nil
}

func (d DummyCollector) ObserveCounter(name string, inc int, labels map[string]string) error {
	return nil
}

