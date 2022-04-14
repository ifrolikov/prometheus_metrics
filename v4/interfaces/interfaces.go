package interfaces

import "time"

type Collector interface {
	ObserveTimer(name string, startTime time.Time, labels map[string]string) error
	ObserveHistogram(name string, startTime time.Time, labels map[string]string) error
	ObserveCounter(name string, inc int, labels map[string]string) error
	ObserveGauge(name string, inc int, labels map[string]string) error
}