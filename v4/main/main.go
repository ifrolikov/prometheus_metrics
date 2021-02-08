package main

import (
	"context"
	"github.com/ifrolikov/prometheus_metrics/v4/grafana"
)

func main() {
	grafanaService := grafana.NewService(
		"https://grafana.monitoring.devel.tutu.ru",
		"eyJrIjoibHdERFpQQnAzTmFjbzNyd1c3WDFERFlyQXNINHV1NjgiLCJuIjoiZnJvbGlrb3YiLCJpZCI6MX0=",
		"openshift-prod-10s",
	)

	ctx := context.TODO()
	err := grafanaService.PushTimerGraph("Frolikov Dashboard From API",
		"aviaapi_graphql_total_processing_time",
		"Тайминг запросов в монолит для GraphQL",
		"aviaapi",
		"public",
		ctx,
		nil)
	if err != nil {
		panic(err)
	}

	datasource := "paas-production-10s"
	err = grafanaService.PushCustomCounterGraph("Frolikov Dashboard From API",
		`statsd_avia_api_endpoint_process_count{ednpoint="RefundCalculate",status="error"}`,
		"К-во запросов на расчет возврата в монолит в час",
		ctx,
		&datasource)
	if err != nil {
		panic(err)
	}
}
