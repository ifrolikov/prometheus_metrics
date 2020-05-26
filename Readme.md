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
    prometheus_metrics.InitGlobalCollector("pod name","service namespace", "service subsystem")
        
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
	)

	ctx := context.TODO()
	err := grafanaService.PushTimerGraph("Frolikov Dashboard From API",
		"aviaapi_graphql_total_processing_time",
		"Тайминг запросов в монолит для GraphQL",
		"aviaapi",
		"public",
		ctx)
	if err != nil {
		panic(err)
	}

	grafanaService.SetPaasDatasource()
	err = grafanaService.PushCustomCounterGraph("Frolikov Dashboard From API",
		`statsd_avia_api_endpoint_process_count{ednpoint="RefundCalculate",status="error"}`,
		"К-во запросов на расчет возврата в монолит в час",
		ctx)
	if err != nil {
		panic(err)
	}
}

```