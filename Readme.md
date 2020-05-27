# Prometheus metric tools
## Observe timer metrics or counter metric

```go
package main

import (
	"github.com/ifrolikov/prometheus_metrics"
	"time"
)

func main() {
    startTime := time.Now()
    prometheus_metrics.InitGlobalCollector("pod name","service namespace", "service subsystem", nil)
        
    metricCollector := prometheus_metrics.GetGlobalCollector()
    
    err := metricCollector.ObserveCounter("full_counter_metric_name", 1, map[string]string{
    	"label_name": "label val",
    })
    if err != nil {
    	// log error or something else
    }
    
    err = metricCollector.ObserveTimer("full_timer_metric_name", startTime, map[string]string{
    	"first_label": "label val 1",
    	"second_label": "label val 2",
    })
    if err != nil {
    	// log error or something else
    }
}
```

If you want to create grafana metrics:

```go
package main

import (
	"github.com/ifrolikov/prometheus_metrics"
	"github.com/ifrolikov/prometheus_metrics/grafana"
	"time"
)

func main() {
    startTime := time.Now()
    prometheus_metrics.InitGlobalCollector("pod name",
    	"service namespace", 
    	"service subsystem",
    	&grafana.InitData{
    		"grafana api url",
    		"grafana api brearer auth key",
    		"default dashboard title",
    		"default datasource name",
    	})
        
    metricCollector := prometheus_metrics.GetGlobalCollector()
    
    err := metricCollector.ObserveCounter("full_counter_metric_name", 1, map[string]string{
    	"label_name": "label val",
    	string(grafana.LABEL_TITLE): "your graph title",
    	string(grafana.LABEL_DASHBOARD_TITLE): "custom dashboard title", // additionally, not required
    	string(grafana.LABEL_DATASOURCE): "custom datasource name", // additionally, not required
    })
    if err != nil {
    	// log error or something else
    }
}
```

## Push timer and counter metric to grafana

```go
package main

import (
	"context"
	"github.com/ifrolikov/prometheus_metrics/grafana"
)

func main() {
	grafanaService := grafana.NewService(
		"http://grafana.api",
		"BREARER_AUTH_KEY",
		"default datasource",
	)

	ctx := context.TODO()
	err := grafanaService.PushTimerGraph("Frolikov Dashboard From API",
		"metric_name",
		"timer name for grafana",
		"your_service_namespace",
		"your_service_subtype",
		ctx,
		nil)
	if err != nil {
		panic(err)
	}

    datasource := "you datasource"
	err = grafanaService.PushCustomCounterGraph("Frolikov Dashboard From API",
		`metric_name`,
		"counter name for grafana",
		ctx,
		&datasource)
	if err != nil {
		panic(err)
	}
}

```