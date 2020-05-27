package prometheus_metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ifrolikov/prometheus_metrics/grafana"
	"github.com/prometheus/client_golang/prometheus"
	"sort"
	"sync"
	"time"
)

const (
	GRAPH_TYPE_COUNTER graphType = "counter"
	GRAPH_TYPE_TIMER   graphType = "timer"
)

var globalCollector *Collector

type graphType string

type Collector struct {
	podName                 string
	timeMetricsMap          map[string]*prometheus.SummaryVec
	timeMetricsLabelsMap    map[string][]string
	counterMetricsMap       map[string]*prometheus.CounterVec
	counterMetricsLabelsMap map[string][]string
	namespace               string
	subsystem               string
	mtx                     *sync.Mutex
	grafanaService          *grafana.Service
	grafanaDashboard        *string
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
			grafanaInitData.DefaultDashboard)
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

func (this *Collector) ObserveTimer(name string, startTime time.Time, labels map[string]string) error {
	defer this.mtx.Unlock()
	this.mtx.Lock()

	specialLabels := map[string]string{}
	if labels == nil {
		labels = map[string]string{}
	} else {
		labels, specialLabels = this.separateSpecialLabels(labels)
	}

	err := this.initTimerIfNotExist(name, labels)
	if err != nil {
		return err
	}
	this.timeMetricsMap[name].With(labels).Observe(float64(time.Since(startTime)))

	err = this.tryToCreateGrafanaGraph(specialLabels, GRAPH_TYPE_TIMER, name, context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func (this *Collector) ObserveCounter(name string, inc int, labels map[string]string) error {
	defer this.mtx.Unlock()
	this.mtx.Lock()

	specialLabels := map[string]string{}
	if labels == nil {
		labels = map[string]string{}
	} else {
		labels, specialLabels = this.separateSpecialLabels(labels)
	}

	err := this.initCounterIfNotExist(name, labels)
	if err != nil {
		return err
	}
	this.counterMetricsMap[name].With(labels).Add(float64(inc))

	err = this.tryToCreateGrafanaGraph(specialLabels, GRAPH_TYPE_COUNTER, name, context.TODO())
	if err != nil {
		return err
	}
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

func (this *Collector) separateSpecialLabels(labels map[string]string) (map[string]string, map[string]string) {
	specialLabels := map[string]string{}
	customLabels := map[string]string{}

	for key, val := range labels {
		isSpecialLabel := false
		for label, _ := range grafana.GrafanaLabels {
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

func (this *Collector) tryToCreateGrafanaGraph(labels map[string]string,
	sendMethodType graphType,
	metricName string,
	ctx context.Context) error {
	if graphTitle, ok := labels[string(grafana.LABEL_TITLE)]; ok {
		if this.grafanaService == nil {
			return errors.New("grafana service is not initialized")
		}
		dashboard, isCustomDashboard := labels[string(grafana.LABEL_DASHBOARD_TITLE)]
		if !isCustomDashboard {
			dashboard = *this.grafanaDashboard
		}
		var datasource *string
		if val, isCustomDatasource := labels[string(grafana.LABEL_DASHBOARD_TITLE)]; isCustomDatasource {
			datasource = &val
		}

		switch sendMethodType {
		case GRAPH_TYPE_COUNTER:
			return this.grafanaService.PushCounterGraph(dashboard, metricName, graphTitle, this.namespace, this.subsystem, ctx, datasource)
		case GRAPH_TYPE_TIMER:
			break
		default:
			return errors.New(fmt.Sprintf("unknown graph type: %s", string(sendMethodType)))
		}
	}
	return nil
}
