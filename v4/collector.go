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
	podName                   string
	timeMetricsMap            map[string]*prometheus.SummaryVec
	timeMetricsLabelsMap      map[string][]string
	counterMetricsMap         map[string]*prometheus.CounterVec
	gaugeMetricsMap           map[string]*prometheus.GaugeVec
	counterMetricsLabelsMap   map[string][]string
	namespace                 string
	subsystem                 string
	mtx                       *sync.Mutex
	grafanaDashboard          *string
	histogramMetricsMap       map[string]*prometheus.HistogramVec
	histogramMetricsLabelsMap map[string][]string
	gaugeMetricsLabelsMap     map[string][]string
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

func (c *Collector) ObserveTimer(name string, startTime time.Time, labels map[string]string) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	err := c.initTimerIfNotExist(name, labels)
	if err != nil {
		return err
	}
	c.timeMetricsMap[name].With(labels).Observe(float64(time.Since(startTime)))

	return nil
}

func (c *Collector) ObserveHistogram(name string, startTime time.Time, labels map[string]string) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	err := c.initHistogramIfNotExist(name, labels)
	if err != nil {
		return err
	}
	c.histogramMetricsMap[name].With(labels).Observe(time.Since(startTime).Seconds())

	//todo push grafana graph
	return nil
}

func (c *Collector) ObserveCounter(name string, inc int, labels map[string]string) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	err := c.initCounterIfNotExist(name, labels)
	if err != nil {
		return err
	}
	c.counterMetricsMap[name].With(labels).Add(float64(inc))

	return nil
}

func (c *Collector) ObserveGauge(name string, inc int, labels map[string]string) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	err := c.initGaugeIfNotExist(name, labels)
	if err != nil {
		return err
	}
	c.gaugeMetricsMap[name].With(labels).Set(float64(inc))
	return nil
}

func (c *Collector) initTimerIfNotExist(name string, labels map[string]string) error {
	if c.timeMetricsMap == nil {
		c.timeMetricsMap = make(map[string]*prometheus.SummaryVec)
		c.timeMetricsLabelsMap = make(map[string][]string)
	}

	var labelNames []string
	if labels != nil {
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
	}
	sort.Strings(labelNames)

	if _, ok := c.timeMetricsMap[name]; !ok {
		c.timeMetricsMap[name] = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:   c.namespace,
				Subsystem:   c.subsystem,
				Name:        name,
				Help:        "dynamic metric " + name,
				Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
				ConstLabels: map[string]string{"podname": c.podName},
			}, labelNames)
		c.timeMetricsLabelsMap[name] = labelNames
		prometheus.MustRegister(c.timeMetricsMap[name])
	} else {
		marshaledCurrentMetricLabels, _ := json.Marshal(c.timeMetricsLabelsMap[name])
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

func (c *Collector) initHistogramIfNotExist(name string, labels map[string]string) error {
	if c.histogramMetricsMap == nil {
		c.histogramMetricsMap = make(map[string]*prometheus.HistogramVec)
		c.histogramMetricsLabelsMap = make(map[string][]string)
	}

	var labelNames []string
	if labels != nil {
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
	}
	sort.Strings(labelNames)

	if _, ok := c.histogramMetricsMap[name]; !ok {
		c.histogramMetricsMap[name] = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   c.namespace,
				Subsystem:   c.subsystem,
				Name:        name,
				Help:        "dynamic metric " + name,
				Buckets:     []float64{.1, .25, .5, .75, .85, 1, 1.5, 2, 2.5, 3, 4, 6, 8, 10},
				ConstLabels: map[string]string{"podname": c.podName},
			}, labelNames)
		c.histogramMetricsLabelsMap[name] = labelNames
		prometheus.MustRegister(c.histogramMetricsMap[name])
	} else {
		marshaledCurrentMetricLabels, _ := json.Marshal(c.histogramMetricsLabelsMap[name])
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

func (c *Collector) initCounterIfNotExist(name string, labels map[string]string) error {
	if c.counterMetricsMap == nil {
		c.counterMetricsMap = make(map[string]*prometheus.CounterVec)
		c.counterMetricsLabelsMap = make(map[string][]string)
	}

	var labelNames []string
	if labels != nil {
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
	}
	sort.Strings(labelNames)

	if _, ok := c.counterMetricsMap[name]; !ok {
		c.counterMetricsMap[name] = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   c.namespace,
				Subsystem:   c.subsystem,
				Name:        name,
				Help:        "dynamic metric " + name,
				ConstLabels: map[string]string{"podname": c.podName},
			}, labelNames)
		c.counterMetricsLabelsMap[name] = labelNames
		prometheus.MustRegister(c.counterMetricsMap[name])
	} else {
		marshaledCurrentMetricLabels, _ := json.Marshal(c.counterMetricsLabelsMap[name])
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

func (c *Collector) initGaugeIfNotExist(name string, labels map[string]string) error {
	if c.gaugeMetricsMap == nil {
		c.gaugeMetricsMap = make(map[string]*prometheus.GaugeVec)
		c.gaugeMetricsLabelsMap = make(map[string][]string)
	}

	var labelNames []string
	if labels != nil {
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
	}
	sort.Strings(labelNames)

	if _, ok := c.gaugeMetricsMap[name]; !ok {
		c.gaugeMetricsMap[name] = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   c.namespace,
				Subsystem:   c.subsystem,
				Name:        name,
				Help:        "dynamic metric " + name,
				ConstLabels: map[string]string{"podname": c.podName},
			}, labelNames)
		c.gaugeMetricsLabelsMap[name] = labelNames
		prometheus.MustRegister(c.gaugeMetricsMap[name])
	} else {
		marshaledCurrentMetricLabels, _ := json.Marshal(c.gaugeMetricsLabelsMap[name])
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
