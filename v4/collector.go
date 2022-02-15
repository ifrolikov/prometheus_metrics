package prometheus_metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ifrolikov/prometheus_metrics/v4/grafana"
	"github.com/prometheus/client_golang/prometheus"
	"sort"
	"sync"
	"time"
)

const (
	GRAPH_TYPE_HISTOGRAM graphType = "histogram"
	GRAPH_TYPE_COUNTER graphType = "counter"
	GRAPH_TYPE_TIMER   graphType = "timer"
)

var globalCollector *Collector

type graphType string

type Collector struct {
	podName                   string
	timeMetricsMap            map[string]*prometheus.SummaryVec
	timeMetricsLabelsMap      map[string][]string
	counterMetricsMap         map[string]*prometheus.CounterVec
	counterMetricsLabelsMap   map[string][]string
	namespace                 string
	subsystem                 string
	mtx                       *sync.Mutex
	grafanaService            *grafana.Service
	grafanaDashboard          *string
	histogramMetricsMap       map[string]*prometheus.HistogramVec
	histogramMetricsLabelsMap map[string][]string
}

func NewCollector(podName string, namespace string, subsystem string, grafanaInitData *grafana.InitData) *Collector {
	collector := &Collector{
		podName:   podName,
		namespace: namespace,
		subsystem: subsystem,
		mtx:       &sync.Mutex{},
	}
	if grafanaInitData != nil {
		collector.grafanaService = grafana.NewService(
			grafanaInitData.ApiUrl,
			grafanaInitData.AuthKey,
			grafanaInitData.DataSource)
		collector.grafanaDashboard = &grafanaInitData.DefaultDashboard
	}
	return collector
}

func GetGlobalCollector() *Collector {
	return globalCollector
}

func InitGlobalCollector(podName string, namespace string, subsystem string, grafanaInitData *grafana.InitData) *Collector {
	globalCollector = NewCollector(podName, namespace, subsystem, grafanaInitData)
	return globalCollector
}

func (c *Collector) ObserveTimer(name string, startTime time.Time, labels map[string]string) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	specialLabels := map[string]string{}
	if labels == nil {
		labels = map[string]string{}
	} else {
		labels, specialLabels = c.separateSpecialLabels(labels)
	}

	err := c.initTimerIfNotExist(name, labels)
	if err != nil {
		return err
	}
	c.timeMetricsMap[name].With(labels).Observe(float64(time.Since(startTime)))

	err = c.tryToCreateGrafanaGraph(specialLabels, GRAPH_TYPE_TIMER, name, context.TODO())
	if err != nil {
		return err
	}
	return nil
}


func (c *Collector) ObserveHistogram(name string, startTime time.Time, labels map[string]string) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	if labels == nil {
		labels = map[string]string{}
	} else {
		labels, _ = c.separateSpecialLabels(labels)
	}

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

	specialLabels := map[string]string{}
	if labels == nil {
		labels = map[string]string{}
	} else {
		labels, specialLabels = c.separateSpecialLabels(labels)
	}

	err := c.initCounterIfNotExist(name, labels)
	if err != nil {
		return err
	}
	c.counterMetricsMap[name].With(labels).Add(float64(inc))

	err = c.tryToCreateGrafanaGraph(specialLabels, GRAPH_TYPE_COUNTER, name, context.TODO())
	if err != nil {
		return err
	}
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
				Buckets:  []float64{.1, .25, .5, .75, .85, 1, 1.5, 2, 2.5, 3, 4, 6, 8, 10},
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

func (c *Collector) separateSpecialLabels(labels map[string]string) (map[string]string, map[string]string) {
	specialLabels := map[string]string{}
	customLabels := map[string]string{}

	for key, val := range labels {
		isSpecialLabel := false
		for _, label := range grafana.GrafanaLabels {
			if string(label) == key {
				specialLabels[key] = val
				isSpecialLabel = true
				break
			}
		}
		if !isSpecialLabel {
			customLabels[key] = val
		}
	}
	return customLabels, specialLabels
}

func (c *Collector) tryToCreateGrafanaGraph(labels map[string]string,
	sendMethodType graphType,
	metricName string,
	ctx context.Context) error {
	if graphTitle, ok := labels[string(grafana.LABEL_TITLE)]; ok {
		if c.grafanaService == nil {
			return errors.New("grafana service is not initialized")
		}
		dashboard, isCustomDashboard := labels[string(grafana.LABEL_DASHBOARD_TITLE)]
		if !isCustomDashboard {
			dashboard = *c.grafanaDashboard
		}
		var datasource *string
		if val, isCustomDatasource := labels[string(grafana.LABEL_DASHBOARD_TITLE)]; isCustomDatasource {
			datasource = &val
		}

		switch sendMethodType {
		case GRAPH_TYPE_COUNTER:
			return c.grafanaService.PushCounterGraph(dashboard, metricName, graphTitle, c.namespace, c.subsystem, ctx, datasource)
		case GRAPH_TYPE_TIMER:
			return c.grafanaService.PushTimerGraph(dashboard, metricName, graphTitle, c.namespace, c.subsystem, ctx, datasource)
		default:
			return errors.New(fmt.Sprintf("unknown graph type: %s", string(sendMethodType)))
		}
	}
	return nil
}
