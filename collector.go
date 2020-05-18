package prometheus_metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var globalCollector *Collector

type Collector struct {
	podName           string
	dynamicMetricsMap map[string]*prometheus.SummaryVec
	namespace         string
	subsystem         string
}

func NewCollector(podName string, namespace string, subsystem string) *Collector {
	collector := &Collector{podName: podName, namespace: namespace, subsystem: subsystem}
	return collector
}

func GetGlobalCollector() *Collector {
	return globalCollector
}

func InitGlobalCollector(podName string, namespace string, subsystem string) *Collector {
	globalCollector = NewCollector(podName, namespace, subsystem)
	return globalCollector
}

func (this *Collector) ObserveDynamicMetric(name string, startTime time.Time) {
	this.initMetricIfNotExist(name)
	labels := map[string]string{}
	this.dynamicMetricsMap[name].With(labels).Observe(float64(time.Since(startTime)))
}

func (this *Collector) initMetricIfNotExist(name string) {
	if this.dynamicMetricsMap == nil {
		this.dynamicMetricsMap = make(map[string]*prometheus.SummaryVec)
	}
	if _, ok := this.dynamicMetricsMap[name]; !ok {
		this.dynamicMetricsMap[name] = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:   this.namespace,
				Subsystem:   this.subsystem,
				Name:        name,
				Help:        "dynamic metric " + name,
				Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
				ConstLabels: map[string]string{"podname": this.podName},
			}, []string{})
		prometheus.MustRegister(this.dynamicMetricsMap[name])
	}
}
