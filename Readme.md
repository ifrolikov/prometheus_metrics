#Prometheus metric team A tools
##Observe timer metrics or counter metric

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