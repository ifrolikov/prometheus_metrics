package prometheus_metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sort"
	"sync"
	"time"
)

var globalCollector *Collector

type Collector struct {
	podName                 string
	timeMetricsMap          map[string]*prometheus.SummaryVec
	timeMetricsLabelsMap    map[string][]string
	counterMetricsMap       map[string]*prometheus.CounterVec
	counterMetricsLabelsMap map[string][]string
	namespace               string
	subsystem               string
	mtx                     *sync.Mutex
}

func NewCollector(podName string, namespace string, subsystem string) *Collector {
	collector := &Collector{
		podName:   podName,
		namespace: namespace,
		subsystem: subsystem,
		mtx:       &sync.Mutex{},
	}
	return collector
}

func GetGlobalCollector() *Collector {
	return globalCollector
}

func InitGlobalCollector(podName string, namespace string, subsystem string) *Collector {
	globalCollector = NewCollector(podName, namespace, subsystem)
	return globalCollector
}

func (this *Collector) ObserveTimer(name string, startTime time.Time, labels map[string]string) error {
	defer this.mtx.Unlock()
	this.mtx.Lock()

	if labels == nil {
		labels = map[string]string{}
	}

	err := this.initTimerIfNotExist(name, labels)
	if err != nil {
		return err
	}
	this.timeMetricsMap[name].With(labels).Observe(float64(time.Since(startTime)))
	return nil
}

func (this *Collector) ObserveCounter(name string, inc int, labels map[string]string) error {
	defer this.mtx.Unlock()
	this.mtx.Lock()

	if labels == nil {
		labels = map[string]string{}
	}

	err := this.initCounterIfNotExist(name, labels)
	if err != nil {
		return err
	}
	this.counterMetricsMap[name].With(labels).Add(float64(inc))
	return nil
}

func (this *Collector) initTimerIfNotExist(name string, labels map[string]string) error {
	if this.timeMetricsMap == nil {
		this.timeMetricsMap = make(map[string]*prometheus.SummaryVec)
		this.timeMetricsLabelsMap = make(map[string][]string)
	}

	labelNames := []string{}
	if labels != nil {
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
	}
	sort.Strings(labelNames)

	if _, ok := this.timeMetricsMap[name]; !ok {
		this.timeMetricsMap[name] = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:   this.namespace,
				Subsystem:   this.subsystem,
				Name:        name,
				Help:        "dynamic metric " + name,
				Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
				ConstLabels: map[string]string{"podname": this.podName},
			}, labelNames)
		this.timeMetricsLabelsMap[name] = labelNames
		prometheus.MustRegister(this.timeMetricsMap[name])
	} else {
		marshaledCurrentMetricLabels, _ := json.Marshal(this.timeMetricsLabelsMap[name])
		marshaledRequestedMetricLabels, _ := json.Marshal(labelNames)
		if string(marshaledCurrentMetricLabels) != string(marshaledRequestedMetricLabels) {
			return errors.New(fmt.Sprintf("invalid metric labels:\n"+
				"current labels: %s\n"+
				"requested labels: %s",
				marshaledCurrentMetricLabels,
				marshaledRequestedMetricLabels))
		}
	}
	return nil
}

func (this *Collector) initCounterIfNotExist(name string, labels map[string]string) error {
	if this.counterMetricsMap == nil {
		this.counterMetricsMap = make(map[string]*prometheus.CounterVec)
		this.counterMetricsLabelsMap = make(map[string][]string)
	}

	labelNames := []string{}
	if labels != nil {
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
	}
	sort.Strings(labelNames)

	if _, ok := this.counterMetricsMap[name]; !ok {
		this.counterMetricsMap[name] = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   this.namespace,
				Subsystem:   this.subsystem,
				Name:        name,
				Help:        "dynamic metric " + name,
				ConstLabels: map[string]string{"podname": this.podName},
			}, labelNames)
		this.counterMetricsLabelsMap[name] = labelNames
		prometheus.MustRegister(this.counterMetricsMap[name])
	} else {
		marshaledCurrentMetricLabels, _ := json.Marshal(this.counterMetricsLabelsMap[name])
		marshaledRequestedMetricLabels, _ := json.Marshal(labelNames)
		if string(marshaledCurrentMetricLabels) != string(marshaledRequestedMetricLabels) {
			return errors.New(fmt.Sprintf("invalid metric labels:\n"+
				"current labels: %s\n"+
				"requested labels: %s",
				marshaledCurrentMetricLabels,
				marshaledRequestedMetricLabels))
		}
	}
	return nil
}
